// Package pve is a minimal read-only client for the Proxmox VE REST API.
package pve

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/MontyNau/lazypve/internal/config"
)

type Client struct {
	host       string
	authHeader string
	httpClient *http.Client
}

func NewClient(cluster config.Cluster) *Client {
	host := strings.TrimRight(cluster.Host, "/")

	return &Client{
		host:       host,
		authHeader: fmt.Sprintf("PVEAPIToken=%s=%s", cluster.TokenID, cluster.TokenSecret),
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: cluster.InsecureSkipVerify}, //nolint:gosec // opt-in for PVE's default self-signed cert
			},
		},
	}
}

func get[T any](ctx context.Context, c *Client, path string) (T, error) {
	var zero T

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.host+"/api2/json"+path, nil)
	if err != nil {
		return zero, err
	}
	req.Header.Set("Authorization", c.authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return zero, fmt.Errorf("pve: %s returned %d: %s", path, resp.StatusCode, string(body))
	}

	var out response[T]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return zero, fmt.Errorf("pve: decoding response from %s: %w", path, err)
	}

	return out.Data, nil
}

func (c *Client) GetNodes(ctx context.Context) ([]Node, error) {
	return get[[]Node](ctx, c, "/nodes")
}

func (c *Client) GetVMs(ctx context.Context, node string) ([]VM, error) {
	return get[[]VM](ctx, c, "/nodes/"+node+"/qemu")
}

func (c *Client) GetContainers(ctx context.Context, node string) ([]Container, error) {
	return get[[]Container](ctx, c, "/nodes/"+node+"/lxc")
}
