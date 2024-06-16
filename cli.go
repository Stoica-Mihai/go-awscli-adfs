package main

import (
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

// Creates a new CLI which will return the values of set flags.
func NewCLI() (username, password, expirationFormatted string, expiration *int32) {
	var expirationSec int

	app := &cli.App{
		Name:  "CLI Access Using SAML",
		Usage: "Generate a credential file locally, which will be stored at /.aws/credentials",
		Action: func(ctx *cli.Context) error {
			return nil
		},
		After: func(ctx *cli.Context) error {
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "username",
				Aliases:     []string{"u"},
				Usage:       "`username` for which the credentials are generated",
				Destination: &username,
				Required:    true,
				Category:    "Mandatory",
			},
			&cli.BoolFlag{
				Name:     "password",
				Aliases:  []string{"p"},
				Usage:    "`password` of the username",
				Category: "Mandatory",
				Action: func(ctx *cli.Context, b bool) error {
					if ctx.Bool("password") {
						fmt.Println("Enter your password:")
						pwd, err := term.ReadPassword(int(os.Stdin.Fd()))

						if err != nil {
							panic(err)
						}

						password = string(pwd)
					}

					return nil
				},
			},
			&cli.IntFlag{
				Name:        "expiration",
				Aliases:     []string{"e"},
				Value:       3600,
				Usage:       "Assign the credentials expiration `time` in seconds",
				Destination: &expirationSec,
				Required:    false,
				Category:    "Optional",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}

	expirationFormatted = convertExpirationTime(&expirationSec)
	expirationPointer := int32(expirationSec)
	expiration = &expirationPointer

	return username, password, expirationFormatted, expiration
}

// Function computes the time as string in the future starting from now + expiration
// in DateTime format
func convertExpirationTime(expiration *int) (futureTime string) {
	futureTime = time.Now().Add(time.Second * time.Duration(*expiration)).Format(time.DateTime)
	return futureTime
}
