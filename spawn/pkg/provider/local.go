package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/scttfrdmn/mycelium/spawn/pkg/config"
)

// LocalProvider implements Provider for local (non-EC2) systems
type LocalProvider struct {
	identity   *Identity
	config     *Config
	configPath string
}

// NewLocalProvider creates a local provider
func NewLocalProvider(ctx context.Context) (*LocalProvider, error) {
	// Load local config
	configPath := os.Getenv("SPAWN_CONFIG")
	localConfig, err := config.LoadLocalConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load local config: %w", err)
	}

	// Get public IP
	publicIP := localConfig.PublicIP
	if publicIP == "" || publicIP == "auto" {
		publicIP = getPublicIP()
	}

	// Get private IP
	privateIP := localConfig.PrivateIP
	if privateIP == "" || privateIP == "auto" {
		privateIP = getPrivateIP()
	}

	identity := &Identity{
		InstanceID: localConfig.InstanceID,
		Region:     localConfig.Region,
		AccountID:  localConfig.AccountID,
		PublicIP:   publicIP,
		PrivateIP:  privateIP,
		Provider:   "local",
	}

	providerConfig := &Config{
		TTL:             config.ParseDuration(localConfig.TTL),
		IdleTimeout:     config.ParseDuration(localConfig.IdleTimeout),
		HibernateOnIdle: localConfig.HibernateOnIdle,
		IdleCPUPercent:  localConfig.IdleCPUPercent,
		OnComplete:      localConfig.OnComplete,
		CompletionFile:  localConfig.CompletionFile,
		CompletionDelay: config.ParseDuration(localConfig.CompletionDelay),
		DNSName:         localConfig.DNS.Name,
		JobArrayID:      localConfig.JobArray.ID,
		JobArrayName:    localConfig.JobArray.Name,
	}

	return &LocalProvider{
		identity:   identity,
		config:     providerConfig,
		configPath: configPath,
	}, nil
}

func (p *LocalProvider) GetIdentity(ctx context.Context) (*Identity, error) {
	return p.identity, nil
}

func (p *LocalProvider) GetConfig(ctx context.Context) (*Config, error) {
	return p.config, nil
}

func (p *LocalProvider) Terminate(ctx context.Context, reason string) error {
	log.Printf("Local instance exiting (reason: %s)", reason)
	// Local mode: just exit process
	// Give a moment for logs to flush
	time.Sleep(1 * time.Second)
	os.Exit(0)
	return nil
}

func (p *LocalProvider) Stop(ctx context.Context, reason string) error {
	// Local instances don't support stop - same as terminate
	log.Printf("Local instance doesn't support stop, exiting instead (reason: %s)", reason)
	return p.Terminate(ctx, reason)
}

func (p *LocalProvider) Hibernate(ctx context.Context) error {
	// Local instances don't support hibernate - same as terminate
	log.Printf("Local instance doesn't support hibernate, exiting instead")
	return p.Terminate(ctx, "hibernate not supported")
}

func (p *LocalProvider) DiscoverPeers(ctx context.Context, jobArrayID string) ([]PeerInfo, error) {
	if jobArrayID == "" {
		return nil, nil
	}

	log.Printf("Discovering peers for job array: %s (local mode)", jobArrayID)

	// Strategy: Use DynamoDB registry (to be implemented in Phase 2)
	// For Phase 1, use static peers file if configured
	localConfig, err := config.LoadLocalConfig(p.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if localConfig.JobArray.PeersFile != "" {
		// Load peers from static file
		return loadPeersFromFile(localConfig.JobArray.PeersFile)
	}

	// No peers configured
	log.Printf("No peer discovery configured for local mode")
	return nil, nil
}

func (p *LocalProvider) IsSpotInstance(ctx context.Context) bool {
	// Local instances are never Spot
	return false
}

func (p *LocalProvider) CheckSpotInterruption(ctx context.Context) (*InterruptionInfo, error) {
	// Local instances never have Spot interruptions
	return nil, nil
}

func (p *LocalProvider) GetProviderType() string {
	return "local"
}

// getPublicIP queries an external service to get the public IP
func getPublicIP() string {
	// Try multiple services in case one is down
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me",
		"https://icanhazip.com",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err == nil {
				ip := strings.TrimSpace(string(body))
				if net.ParseIP(ip) != nil {
					return ip
				}
			}
		}
	}

	log.Printf("Warning: Could not determine public IP")
	return ""
}

// getPrivateIP gets the local network IP address
func getPrivateIP() string {
	// Get local IP by connecting to a remote address (doesn't actually send data)
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// loadPeersFromFile loads peer information from a JSON file
func loadPeersFromFile(path string) ([]PeerInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read peers file: %w", err)
	}

	var peers []PeerInfo
	if err := json.Unmarshal(data, &peers); err != nil {
		return nil, fmt.Errorf("failed to parse peers file: %w", err)
	}

	log.Printf("✓ Loaded %d peers from %s", len(peers), path)
	return peers, nil
}
