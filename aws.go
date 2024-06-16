package main

import (
	"strings"
)

type AWSRole struct {
	RoleArn      *string
	PrincipalARN *string
}

type AWSProfile struct {
	MainAlias       string
	AccountAlias    []string
	AccessKeyId     string
	SecretAccessKey string
	SessionToken    string
	Region          string
	Expiration      string
	Client          string
}

func NewAWSRole(roleArn string, principalARN string) *AWSRole {
	roleArnTrim := strings.TrimSpace(roleArn)
	principalARNTrim := strings.TrimSpace(principalARN)

	return &AWSRole{
		RoleArn:      &roleArnTrim,
		PrincipalARN: &principalARNTrim,
	}
}

// Create new AWS Profile
//
// Returns AWSProfile pointer
func NewAWSProfile(
	accountAlias []string,
	accesskeyid string,
	secretaccesskey string,
	sessiontoken string,
	expiration string,
) *AWSProfile {
	profileManager := NewProfileManager()

	mainAlias := accountAlias[0]
	clientName, regionName := awsClientAndRegion(mainAlias)

	return &AWSProfile{
		AccountAlias:    accountAlias,
		AccessKeyId:     accesskeyid,
		SecretAccessKey: secretaccesskey,
		SessionToken:    sessiontoken,
		Expiration:      expiration,
		MainAlias:       mainAlias,
		Client:          clientName,
		Region:          profileManager.findClientRealRegion(clientName, regionName),
	}
}

func awsClientAndRegion(alias string) (string, *string) {
	formatAlias := strings.Replace(alias, "aws-ecom-titan", "", 1)
	accountAndRegion := strings.Split(formatAlias, "-")

	return accountAndRegion[0], &accountAndRegion[1]
}
