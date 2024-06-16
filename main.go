package main

func main() {
	profileManager := NewProfileManager()

	configLoc := CreateCredsDirectory()

	username, password, expirationFormatted, expiration := NewCLI()

	user := NewUser(username, password, expiration, expirationFormatted)

	sessionResponse := GetSessionResponse(user)

	samlResponse := GetSamlResponse(sessionResponse)

	awsRoles := GetAWSRoles(samlResponse)

	iamOutput := GetIAMOutput(samlResponse, awsRoles, user, profileManager)

	WriteProfiles(iamOutput, configLoc)
}
