// Package config loads lazypve's Proxmox connection settings from the environment.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	// Host is the Proxmox API endpoint, e.g. "https://192.168.1.10:8006".
	Host string
	// TokenID is the full API token identifier, e.g. "root@pam!lazypve".
	TokenID string
	// TokenSecret is the token's UUID secret.
	TokenSecret string
	// InsecureSkipVerify disables TLS certificate verification, needed for
	// Proxmox's default self-signed certificate.
	InsecureSkipVerify bool
}

func Load() (Config, error) {
	cfg := Config{
		Host:               os.Getenv("LAZYPVE_HOST"),
		TokenID:            os.Getenv("LAZYPVE_TOKEN_ID"),
		TokenSecret:        os.Getenv("LAZYPVE_TOKEN_SECRET"),
		InsecureSkipVerify: os.Getenv("LAZYPVE_INSECURE_SKIP_VERIFY") == "true",
	}

	var missing []string
	if cfg.Host == "" {
		missing = append(missing, "LAZYPVE_HOST")
	}
	if cfg.TokenID == "" {
		missing = append(missing, "LAZYPVE_TOKEN_ID")
	}
	if cfg.TokenSecret == "" {
		missing = append(missing, "LAZYPVE_TOKEN_SECRET")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}
