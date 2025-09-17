//go:build server
// +build server

package config

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/Onyz107/onyrat/internal/logger"
	"gopkg.in/yaml.v3"
)

type serverConfig struct {
	Host       string `yaml:"host"`
	Port       string `yaml:"port"`
	PrivateKey string `yaml:"private_key"`
}

var ServerConfigs *serverConfig

//go:embed server.yaml
var serverYAML []byte

// Load server config
func LoadServerConfig() (*serverConfig, error) {
	var cfg serverConfig
	if err := yaml.Unmarshal(serverYAML, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal server config: %w", err)
	}
	return &cfg, nil
}

func init() {
	var err error

	ServerConfigs, err = LoadServerConfig()
	if err != nil {
		logger.Log.Errorf("failed to load client configurations: %v", err)
		os.Exit(1)
	}
}
