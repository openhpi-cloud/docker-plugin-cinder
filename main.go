package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	_log "log"
	"os"

	"github.com/coreos/go-systemd/v22/activation"
	log "github.com/sirupsen/logrus"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
)

type tConfig struct {
	Debug                       bool   `json:"debug,omitempty"`
	Quiet                       bool   `json:"quiet,omitempty"`
	Prefix                      string `json:"prefix,omitempty"`
	IdentityEndpoint            string `json:"endpoint,omitempty"`
	Username                    string `json:"username,omitempty"`
	Password                    string `json:"password,omitempty"`
	DomainID                    string `json:"domainID,omitempty"`
	DomainName                  string `json:"domainName,omitempty"`
	TenantID                    string `json:"tenantId,omitempty"`
	TenantName                  string `json:"tenantName,omitempty"`
	ApplicationCredentialID     string `json:"applicationCredentialId,omitempty"`
	ApplicationCredentialName   string `json:"applicationCredentialName,omitempty"`
	ApplicationCredentialSecret string `json:"applicationCredentialSecret,omitempty"`
	Region                      string `json:"region,omitempty"`
	MachineID                   string `json:"machineID,omitempty"`
	MountDir                    string `json:"mountDir,omitempty"`
}

func init() {
	_log.SetOutput(io.Discard)

	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	var config tConfig
	var configFile string
	var createConfig bool
	flag.BoolVar(&config.Debug, "debug", false, "Enable debug logging")
	flag.BoolVar(&config.Quiet, "quiet", false, "Only report errors")
	flag.StringVar(&configFile, "config", "", "Path to config file")
	flag.StringVar(&config.Prefix, "prefix", "docker-volume", "")
	flag.StringVar(&config.MountDir, "mountDir", "", "")
	flag.BoolVar(&createConfig, "createConfig", false, "Create config file interactively")
	flag.Parse()

	if configFile == "" {
		configFile = "./cinder.json"
	}

	if createConfig {
		_, err := createConfiguration(configFile)
		if err != nil {
			log.Fatalf("Failed to write configuration file: %s", err)
		}
		log.Info("Configuration file written successfully")
		os.Exit(0)
	}

	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	log.SetOutput(os.Stdout)

	content, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = json.Unmarshal(content, &config)
	if err != nil {
		log.Fatal(err.Error())
	}

	if config.MountDir == "" {
		log.Fatal("No mountDir configured. Abort.")
	}

	if config.Quiet {
		log.SetLevel(log.ErrorLevel)
	}

	if config.Debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Debug("Debug logging enabled")

	if config.IdentityEndpoint == "" {
		log.Fatal("Identity endpoint missing")
	}

	opts := gophercloud.AuthOptions{
		IdentityEndpoint:            config.IdentityEndpoint,
		Username:                    config.Username,
		Password:                    config.Password,
		DomainID:                    config.DomainID,
		DomainName:                  config.DomainName,
		TenantID:                    config.TenantID,
		TenantName:                  config.TenantName,
		ApplicationCredentialID:     config.ApplicationCredentialID,
		ApplicationCredentialName:   config.ApplicationCredentialName,
		ApplicationCredentialSecret: config.ApplicationCredentialSecret,
		AllowReauth:                 true,
	}

	logger := log.WithField("endpoint", opts.IdentityEndpoint)
	logger.Info("Connecting...")

	provider, err := openstack.AuthenticatedClient(context.TODO(), opts)
	if err != nil {
		logger.WithError(err).Fatal(err.Error())
	}

	endpointOpts := gophercloud.EndpointOpts{
		Region: config.Region,
	}

	plugin, err := newPlugin(provider, endpointOpts, &config)

	if err != nil {
		logger.WithError(err).Fatal(err.Error())
	}

	handler := volume.NewHandler(plugin)

	logger.Info("Connected.")

	listeners, err := activation.Listeners()

	if err != nil {
		logger.WithError(err).Error(err.Error())
	}

	if len(listeners) > 0 {
		logger.Debugf("Started with socket activation")
		err = handler.Serve(listeners[0])
	} else {
		err = handler.ServeUnix("cinder", 0)
	}

	if err != nil {
		logger.WithError(err).Fatal(err.Error())
	}
}
