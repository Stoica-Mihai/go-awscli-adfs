package main

import (
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProfileManager struct {
	Default  Profile `yaml:"default"`
	Profile1 Profile `yaml:"profile1"`
	Profile2 Profile `yaml:"profile2"`
}

type Profile struct {
	Value string  `yaml:"value"`
	Alpha *string `yaml:"alpha"`
	Beta  *string `yaml:"beta"`
	Gold  *string `yaml:"gold"`
}

func NewProfileManager() (profiles *ProfileManager) {
	profilesData, err := os.ReadFile("profiles.yaml")

	if err != nil {
		panic("The file 'profiles.yaml' was not found")
	}

	unmarshalErr := yaml.Unmarshal(profilesData, &profiles)

	if unmarshalErr != nil {
		panic("Failed to unmarshal 'profiles.yaml': " + unmarshalErr.Error())
	}

	return profiles
}

// Finds the client and region in the profiles.yaml
//
// If found returns it, if not found returns the default and if error occurs, return fallback region (eu-west-1)
func (pm ProfileManager) findClientRealRegion(client string, region *string) string {

	toTitle := func(substring string) string {
		return strings.ToUpper(string(substring[0])) + substring[1:]
	}

	reflection := reflect.ValueOf(pm)

	if reflectedClient := reflection.FieldByName(toTitle(client)); reflectedClient.IsValid() {
		reflection = reflect.ValueOf(reflectedClient.Interface())
		if reflectedRegion := reflection.FieldByName(toTitle(*region)); !reflectedRegion.IsNil() {
			return reflectedRegion.Elem().String()
		}
	}

	return pm.Default.Value
}
