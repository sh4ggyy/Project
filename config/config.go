package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

var Config Configuration

type Configuration struct {
	PersonalToken string
	ListenURL     string
	URLPathPrefix string
}

func LoadConfiguration(FilePath string) error {
	var err error
	Config = Configuration{}

	confLen := len(FilePath)
	if confLen != 0 {
		err = readFromJSON(FilePath)
	} else {
		return errors.New("FilePath parameter was found to be empty")
	}
	if err != nil {
		return err
	}

	return nil
}

// readFromJSON reads config data from JSON-file
func readFromJSON(configFilePath string) error {
	contents, err := ioutil.ReadFile(filepath.Clean(configFilePath))
	if err == nil {
		reader := bytes.NewBuffer(contents)
		err = json.NewDecoder(reader).Decode(&Config)
	}
	if err != nil {
		fmt.Println("Config error", "Reading configuration from JSON (%s) failed: ", configFilePath, err.Error())
	}
	err = validateParams()
	if err != nil {
		fmt.Println("Config error", "Failed to validate config parameters: ", err.Error())
	} else {
		fmt.Println("Configuration has been read from JSON successfully", configFilePath)
	}

	return err
}

func validateParams() error {
	var missingParams string
	if len(Config.PersonalToken) == 0 {
		missingParams = missingParams + "PersonalToken "
	}
	if len(Config.ListenURL) == 0 {
		missingParams = missingParams + "ListenURL "
	}

	if len(missingParams) > 0 {
		return errors.New("Missing Parameters: " + missingParams)
	}
	return nil
}
