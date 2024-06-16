package main

type User struct {
	username            string
	password            string
	expiration          *int32
	expirationFormatted string
	idpentryurl         string
	maxItems            *int32
}

func NewUser(
	username string,
	password string,
	expiration *int32,
	expirationFormatted string,
) *User {
	maxItems := int32(1)
	return &User{
		username:            username,
		password:            password,
		expiration:          expiration,
		idpentryurl:         "<login adfs url>",
		expirationFormatted: expirationFormatted,
		maxItems:            &maxItems,
	}
}
