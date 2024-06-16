package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/azure/go-ntlmssp"
	"github.com/beevik/etree"
	"github.com/bigkevmcd/go-configparser"
)

var (
	AWSCONFIGFILENAME = "credentials"
	USERAGENT         = "Mozilla/5.0 (compatible, MSIE 11, Windows NT 6.3; Trident/7.0; rv:11.0) like Gecko"
	RULLER            = strings.Repeat("═", 50)
)

// Creates the /.aws/credentials path/file
//
// Returns the full config location
func CreateCredsDirectory() (configLoc string) {
	currentUser, err := user.Current()

	if err != nil {
		panic(err)
	}

	awsDirPath := filepath.Join(currentUser.HomeDir, ".aws")
	configLoc = filepath.Join(awsDirPath, AWSCONFIGFILENAME)

	if _, err := os.Stat(awsDirPath); os.IsNotExist(err) {
		if err := os.Mkdir(awsDirPath, os.ModePerm); err != nil {
			panic(err)
		}
	}

	if _, err := os.Stat(configLoc); errors.Is(err, os.ErrNotExist) {
		if _, err := os.Create(configLoc); err != nil {
			panic(err)
		}
	}

	return configLoc
}

// Makes a request to ADFS and gets the response
//
// Returns the response string
func GetSessionResponse(user *User) string {
	jar, err := cookiejar.New(nil)

	if err != nil {
		panic(err)
	}

	client := &http.Client{
		Transport: ntlmssp.Negotiator{
			RoundTripper: &http.Transport{},
		},
		Jar: jar,
	}

	req, err := http.NewRequest("GET", user.idpentryurl, nil)

	if err != nil {
		panic(err)
	}

	req.SetBasicAuth(user.username, user.password)
	req.Header.Set("User-Agent", USERAGENT)

	res, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(res.Body)

	if err != nil {
		panic(err)
	}

	defer res.Body.Close()

	return string(body)
}

// Querys the session response and gets the SAML response
//
// Returns the SAML response token
func GetSamlResponse(session string) (samlResponse *string) {
	reader := strings.NewReader(session)
	doc, err := goquery.NewDocumentFromReader(reader)

	if err != nil {
		panic(err)
	}

	doc.Find("input").Each(func(i int, s *goquery.Selection) {
		if valueAttr, valueAttrExists := s.Attr("value"); valueAttrExists {
			samlResponse = &valueAttr
		}
	})

	return samlResponse
}

// Converts the SAMLAssertion from base64 to string to etree
// SAML Assertion of type nil will exit the app under the assumption that the user password is incorrect
//
// Returns all aws roles as array
func GetAWSRoles(SAMLAssertion *string) (awsRoles []*AWSRole) {
	if SAMLAssertion == nil {
		fmt.Println("\033[1;31mYour password is incorrect\033[0m")
		os.Exit(1)
	}

	root := etree.NewDocument()
	SAMLBytes, err := base64.StdEncoding.DecodeString(*SAMLAssertion)

	if err != nil {
		panic(err)
	}

	SAMLString := string(SAMLBytes)

	root.ReadFromString(SAMLString)

	for _, role := range root.FindElements("//Attribute[@Name='https://aws.amazon.com/SAML/Attributes/Role']/*") {
		chunks := strings.Split(role.Text(), ",")
		if strings.Contains(chunks[0], "saml-provider") {
			awsRoles = append(awsRoles, NewAWSRole(chunks[1], chunks[0]))
		} else {
			awsRoles = append(awsRoles, NewAWSRole(chunks[0], chunks[1]))
		}
	}

	return awsRoles
}

// Creates a separate goroutine for each role it has to request data from aws
//
// Returns an array with all the profiles that were successfully retrieved
func GetIAMOutput(SAMLAssertion *string, awsRoles []*AWSRole, user *User, profileManager *ProfileManager) (profiles []*AWSProfile) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	channel := make(chan *AWSProfile, runtime.NumCPU())

	var wg sync.WaitGroup
	wg.Add(len(awsRoles))

	if err != nil {
		panic(err)
	}

	fmt.Println("\033[1;33mGenerating profiles\033[0m")
	// Go routine that reads the messages sent on the channel
	go func() {
		for m := range channel {
			profiles = append(profiles, m)
		}
	}()

	svc := sts.NewFromConfig(cfg)

	for _, awsRole := range awsRoles {

		// Creates a goroutine for each role
		go func(awsRole *AWSRole) {
			defer wg.Done()

			stsResult, err := svc.AssumeRoleWithSAML(
				context.TODO(),
				&sts.AssumeRoleWithSAMLInput{
					PrincipalArn:    awsRole.PrincipalARN,
					RoleArn:         awsRole.RoleArn,
					DurationSeconds: user.expiration,
					SAMLAssertion:   SAMLAssertion,
				},
				func(o *sts.Options) {
					o.Region = profileManager.Default.Value
				},
			)

			if err != nil {
				return
			}

			credentials := credentials.NewStaticCredentialsProvider(
				*stsResult.Credentials.AccessKeyId,
				*stsResult.Credentials.SecretAccessKey,
				*stsResult.Credentials.SessionToken,
			)

			iamClient := iam.New(iam.Options{
				Region:      profileManager.Default.Value,
				Credentials: credentials,
			})

			iamOutput, err := iamClient.ListAccountAliases(context.TODO(), &iam.ListAccountAliasesInput{
				MaxItems: user.maxItems,
			})

			if err != nil {
				return
			}

			channel <- NewAWSProfile(
				iamOutput.AccountAliases,
				credentials.Value.AccessKeyID,
				credentials.Value.SecretAccessKey,
				credentials.Value.SessionToken,
				user.expirationFormatted,
			)

		}(awsRole)
	}

	wg.Wait()
	close(channel)
	return profiles
}

// Writes the profiles returned by GetIAMOutput to the credentials location
func WriteProfiles(iamOutput []*AWSProfile, configLoc string) {
	var output string

	config, err := configparser.NewConfigParserFromFile(configLoc)

	if err != nil {
		panic(err)
	}

	for _, profile := range iamOutput {
		config.AddSection(profile.MainAlias)
		config.Set(profile.MainAlias, "aws_access_key_id", profile.AccessKeyId)
		config.Set(profile.MainAlias, "aws_secret_access_key", profile.SecretAccessKey)
		config.Set(profile.MainAlias, "aws_session_token", profile.SessionToken)
		config.Set(profile.MainAlias, "region", profile.Region)
		config.SaveWithDelimiter(configLoc, "=")

		// Writes to output information about the config
		output += RULLER + "\n" + fmt.Sprintf("Profile: %s \nExpiration: %s \nRegion: %s", profile.MainAlias, profile.Expiration, profile.Region) + "\n" + RULLER + "\n"
	}

	fmt.Println(output)
}
