package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/charmbracelet/huh"
)

type setupConfig struct {
	IdentityEndpoint            string `json:"endpoint,omitempty"`
	ApplicationCredentialID     string `json:"applicationCredentialId,omitempty"`
	ApplicationCredentialSecret string `json:"applicationCredentialSecret,omitempty"`
	Region                      string `json:"region,omitempty"`
	MountDir                    string `json:"mountDir,omitempty"`
}

func createConfiguration(configPath string) error {
	var config setupConfig

	credentialInstructions :=
		`To get an Application Credential ID you must create new Application Credentials
via the web console (Horizon).
Navigate to "Identity > Application Credentials > Create Application Credential"
Enter a descriptive name and optionally a description and expiration date.
Press 'Create Application Credential', copy the displayed ID and paste it here.

Do not close the window yet.`

	mountDirInstructions := `Directory used for mounting cinder volumes
Leave blank for default`

	stat, err := os.Stat(configPath)
	if err == nil {
		if stat.IsDir() {
			return fmt.Errorf("The configuration file path already is a directory. Delete it or choose a different path to continue.")
		} else {
			overwriteFile := false

			err = huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().Title("The config file already exists. Overwrite it?").Description(fmt.Sprintf("Path: '%s'", configPath)).Value(&overwriteFile),
				),
			).Run()
			if err != nil {
				return err
			}
			if !overwriteFile {
				return fmt.Errorf("user aborted")
			}
		}
	} else if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(path.Dir(configPath), 0775)
	}

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Open Stack endpoint URL").Value(&config.IdentityEndpoint),
			huh.NewInput().Title("Open Stack Region Name").Value(&config.Region),
		),
		huh.NewGroup(
			huh.NewInput().Title("Application Credential ID").Value(&config.ApplicationCredentialID).Description(credentialInstructions),
			huh.NewInput().Title("Application Credential Secret").Value(&config.ApplicationCredentialSecret).Description("Copy the displayed Secret and paste it here").EchoMode(huh.EchoModePassword),
		),
		huh.NewGroup(
			huh.NewInput().Title("Mount Dir").Value(&config.MountDir).Description(mountDirInstructions),
		),
	).Run()
	if err != nil {
		return err
	}

	return writeConfigurationFile(config, configPath)
}

func writeConfigurationFile(config setupConfig, path string) error {
	configText, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}

	err = os.WriteFile(path, configText, 0664)
	if err != nil {
		return err
	}
	return nil
}
