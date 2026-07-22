// Package config loads lazypve's Proxmox connection settings, one or more clusters.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Cluster is one Proxmox VE endpoint to connect to. Most setups have exactly
// one; LAZYPVE_CLUSTERS_FILE allows listing several.
type Cluster struct {
	// Name identifies this cluster in the UI. Must be unique across clusters.
	Name string `json:"name"`
	// Host is the Proxmox API endpoint, e.g. "https://192.168.1.10:8006".
	Host string `json:"host"`
	// TokenID is the full API token identifier, e.g. "root@pam!lazypve".
	TokenID string `json:"token_id"`
	// TokenSecret is the token's UUID secret.
	TokenSecret string `json:"token_secret"`
	// InsecureSkipVerify disables TLS certificate verification, needed for
	// Proxmox's default self-signed certificate.
	InsecureSkipVerify bool `json:"insecure_skip_verify"`
}

// Load returns the clusters lazypve should connect to. If LAZYPVE_CLUSTERS_FILE
// is set, clusters are read from that JSON file. Otherwise a single cluster is
// built from LAZYPVE_HOST/LAZYPVE_TOKEN_ID/LAZYPVE_TOKEN_SECRET/LAZYPVE_INSECURE_SKIP_VERIFY
// (and optionally LAZYPVE_NAME), matching the original single-cluster setup.
func Load() ([]Cluster, error) {
	if path := os.Getenv("LAZYPVE_CLUSTERS_FILE"); path != "" {
		return loadFromFile(path)
	}
	return loadSingleFromEnv()
}

func loadSingleFromEnv() ([]Cluster, error) {
	name := os.Getenv("LAZYPVE_NAME")
	if name == "" {
		name = "default"
	}

	c := Cluster{
		Name:               name,
		Host:               os.Getenv("LAZYPVE_HOST"),
		TokenID:            os.Getenv("LAZYPVE_TOKEN_ID"),
		TokenSecret:        os.Getenv("LAZYPVE_TOKEN_SECRET"),
		InsecureSkipVerify: os.Getenv("LAZYPVE_INSECURE_SKIP_VERIFY") == "true",
	}

	var missing []string
	if c.Host == "" {
		missing = append(missing, "LAZYPVE_HOST")
	}
	if c.TokenID == "" {
		missing = append(missing, "LAZYPVE_TOKEN_ID")
	}
	if c.TokenSecret == "" {
		missing = append(missing, "LAZYPVE_TOKEN_SECRET")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return []Cluster{c}, nil
}

func loadFromFile(path string) ([]Cluster, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var clusters []Cluster
	if err := json.Unmarshal(data, &clusters); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(clusters) == 0 {
		return nil, fmt.Errorf("%s defines no clusters", path)
	}

	seen := make(map[string]bool, len(clusters))
	for _, c := range clusters {
		if c.Name == "" {
			return nil, fmt.Errorf("%s: cluster with empty name", path)
		}
		if seen[c.Name] {
			return nil, fmt.Errorf("%s: duplicate cluster name %q", path, c.Name)
		}
		seen[c.Name] = true
	}

	return clusters, nil
}
