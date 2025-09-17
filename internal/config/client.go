//go:build client
// +build client

package config

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/Onyz107/onyrat/internal/logger"
	"gopkg.in/yaml.v3"
)

type clientConfig struct {
	Host      string `yaml:"host"`
	Port      string `yaml:"port"`
	PublicKey string `yaml:"public_key"`
}

var ClientConfigs *clientConfig

//go:embed client.yaml
var clientYAML []byte

// LoadClientConfig parses the embedded client.yaml into a clientConfig struct
func LoadClientConfig() (*clientConfig, error) {
	var cfg clientConfig
	if err := yaml.Unmarshal(clientYAML, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal client config: %w", err)
	}
	return &cfg, nil
}

func init() {
	var err error

	ClientConfigs, err = LoadClientConfig()
	if err != nil {
		logger.Log.Errorf("failed to load client configurations: %v", err)
		os.Exit(1)
	}
}
