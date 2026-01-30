package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// LocalConfig represents the local configuration file format
type LocalConfig struct {
	// Instance identity
	InstanceID string `yaml:"instance_id"`
	Region     string `yaml:"region"`
	AccountID  string `yaml:"account_id"`
	PublicIP   string `yaml:"public_ip"`
	PrivateIP  string `yaml:"private_ip"`

	// Lifecycle configuration
	TTL             string  `yaml:"ttl"`
	IdleTimeout     string  `yaml:"idle_timeout"`
	HibernateOnIdle bool    `yaml:"hibernate_on_idle"`
	IdleCPUPercent  float64 `yaml:"idle_cpu_percent"`
	OnComplete      string  `yaml:"on_complete"`
	CompletionFile  string  `yaml:"completion_file"`
	CompletionDelay string  `yaml:"completion_delay"`

	// DNS configuration (optional)
	DNS struct {
		Enabled bool   `yaml:"enabled"`
		Name    string `yaml:"name"`
		Domain  string `yaml:"domain"`
	} `yaml:"dns"`

	// Job array configuration
	JobArray struct {
		ID        string `yaml:"id"`
		Name      string `yaml:"name"`
		Index     int    `yaml:"index"`
		PeersFile string `yaml:"peers_file"`
	} `yaml:"job_array"`

	// Orchestrator configuration (for burst coordination)
	Orchestrator struct {
		Enabled           bool   `yaml:"enabled"`
		URL               string `yaml:"url"`
		RegisterOnStartup bool   `yaml:"register_on_startup"`
	} `yaml:"orchestrator"`
}

// LoadLocalConfig loads configuration from file and environment variables
func LoadLocalConfig(configPath string) (*LocalConfig, error) {
	// Default config path
	if configPath == "" {
		configPath = "/etc/spawn/local.yaml"
	}

	// Check if config file exists
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config file doesn't exist - use environment variables only
		return loadFromEnv(), nil
	}

	// Parse YAML
	var config LocalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables
	mergeEnvVars(&config)

	// Apply defaults
	applyDefaults(&config)

	return &config, nil
}

// loadFromEnv creates config from environment variables only
func loadFromEnv() *LocalConfig {
	config := &LocalConfig{}
	mergeEnvVars(config)
	applyDefaults(config)
	return config
}

// mergeEnvVars overrides config with environment variables
func mergeEnvVars(config *LocalConfig) {
	if val := os.Getenv("SPAWN_INSTANCE_ID"); val != "" {
		config.InstanceID = val
	}
	if val := os.Getenv("SPAWN_REGION"); val != "" {
		config.Region = val
	}
	if val := os.Getenv("SPAWN_ACCOUNT_ID"); val != "" {
		config.AccountID = val
	}
	if val := os.Getenv("SPAWN_PUBLIC_IP"); val != "" {
		config.PublicIP = val
	}
	if val := os.Getenv("SPAWN_PRIVATE_IP"); val != "" {
		config.PrivateIP = val
	}
	if val := os.Getenv("SPAWN_TTL"); val != "" {
		config.TTL = val
	}
	if val := os.Getenv("SPAWN_IDLE_TIMEOUT"); val != "" {
		config.IdleTimeout = val
	}
	if val := os.Getenv("SPAWN_ON_COMPLETE"); val != "" {
		config.OnComplete = val
	}
	if val := os.Getenv("SPAWN_DNS_NAME"); val != "" {
		config.DNS.Name = val
	}
	if val := os.Getenv("SPAWN_JOB_ARRAY_ID"); val != "" {
		config.JobArray.ID = val
	}
}

// applyDefaults fills in default values
func applyDefaults(config *LocalConfig) {
	if config.InstanceID == "" || config.InstanceID == "auto" {
		// Use hostname as instance ID
		hostname, err := os.Hostname()
		if err == nil {
			config.InstanceID = "local-" + hostname
		} else {
			config.InstanceID = "local-unknown"
		}
	}

	if config.Region == "" {
		config.Region = "local"
	}

	if config.AccountID == "" {
		config.AccountID = "local"
	}

	if config.IdleCPUPercent == 0 {
		config.IdleCPUPercent = 5.0
	}

	if config.OnComplete == "" {
		config.OnComplete = "exit"
	}

	if config.CompletionFile == "" && config.OnComplete != "" {
		config.CompletionFile = "/tmp/SPAWN_COMPLETE"
	}

	if config.CompletionDelay == "" && config.OnComplete != "" {
		config.CompletionDelay = "30s"
	}
}

// ParseDuration parses a duration string, returns zero duration on error
func ParseDuration(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}
