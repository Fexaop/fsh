package appconfig

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	RealtimeModeStrictNoDrop = "strict_no_drop"
	RealtimeModeAllowDrop    = "allow_drop"
)

type Config struct {
	Auth        AuthConfig    `json:"auth"`
}

type AuthConfig struct {
	MiddlewareEnabled bool `json:"middleware_enabled"`
}


func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file failed: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config json failed: %w", err)
	}

	return &cfg, nil
}

func DefaultConfig() Config {
	return Config{
		Auth: AuthConfig{
			MiddlewareEnabled: false,
		},
	}
}


