// Package config loads lazypve's Proxmox connection settings, one or more clusters.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Cluster is one Proxmox VE endpoint to connect to.
type Cluster struct {
	// Name identifies this cluster in the UI.
	Name string
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

// Load returns the clusters lazypve should connect to.
//
// If LAZYPVE_CLUSTER1_HOST is set, clusters are read as a numbered list —
// LAZYPVE_CLUSTER1_*, LAZYPVE_CLUSTER2_*, and so on — stopping at the first
// missing number. Otherwise a single cluster is built from the unnumbered
// LAZYPVE_HOST/LAZYPVE_TOKEN_ID/LAZYPVE_TOKEN_SECRET/LAZYPVE_INSECURE_SKIP_VERIFY
// vars (optionally named via LAZYPVE_NAME, defaulting to "default").
func Load() ([]Cluster, error) {
	if os.Getenv("LAZYPVE_CLUSTER1_HOST") != "" {
		return loadNumberedFromEnv()
	}
	return loadSingleFromEnv()
}

func loadSingleFromEnv() ([]Cluster, error) {
	name := os.Getenv("LAZYPVE_NAME")
	if name == "" {
		name = "default"
	}

	c, err := clusterFromEnv(name, "LAZYPVE_")
	if err != nil {
		return nil, err
	}
	return []Cluster{c}, nil
}

func loadNumberedFromEnv() ([]Cluster, error) {
	var clusters []Cluster
	for i := 1; ; i++ {
		prefix := "LAZYPVE_CLUSTER" + strconv.Itoa(i) + "_"
		if os.Getenv(prefix+"HOST") == "" {
			break
		}

		name := os.Getenv(prefix + "NAME")
		if name == "" {
			name = "cluster" + strconv.Itoa(i)
		}

		c, err := clusterFromEnv(name, prefix)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, c)
	}
	return clusters, nil
}

// clusterFromEnv reads HOST/TOKEN_ID/TOKEN_SECRET/INSECURE_SKIP_VERIFY under
// the given env var prefix (e.g. "LAZYPVE_" or "LAZYPVE_CLUSTER2_").
func clusterFromEnv(name, prefix string) (Cluster, error) {
	hostVar := prefix + "HOST"
	tokenIDVar := prefix + "TOKEN_ID"
	tokenSecretVar := prefix + "TOKEN_SECRET"
	insecureVar := prefix + "INSECURE_SKIP_VERIFY"

	c := Cluster{
		Name:               name,
		Host:               os.Getenv(hostVar),
		TokenID:            os.Getenv(tokenIDVar),
		TokenSecret:        os.Getenv(tokenSecretVar),
		InsecureSkipVerify: os.Getenv(insecureVar) == "true",
	}

	var missing []string
	if c.Host == "" {
		missing = append(missing, hostVar)
	}
	if c.TokenID == "" {
		missing = append(missing, tokenIDVar)
	}
	if c.TokenSecret == "" {
		missing = append(missing, tokenSecretVar)
	}
	if len(missing) > 0 {
		return Cluster{}, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return c, nil
}
