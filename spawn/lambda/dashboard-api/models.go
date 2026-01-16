package main

import (
	"time"
)

// API Response structures

// APIResponse is the standard API response format
type APIResponse struct {
	Success        bool         `json:"success"`
	Message        string       `json:"message,omitempty"`
	Error          string       `json:"error,omitempty"`
	AccountBase36  string       `json:"account_base36,omitempty"`
	RegionsQueried []string     `json:"regions_queried,omitempty"`
	TotalInstances int          `json:"total_instances,omitempty"`
	Instances      []InstanceInfo `json:"instances,omitempty"`
	Instance       *InstanceInfo  `json:"instance,omitempty"`
	User           *UserProfile   `json:"user,omitempty"`
}

// InstanceInfo represents EC2 instance information
type InstanceInfo struct {
	InstanceID           string            `json:"instance_id"`
	Name                 string            `json:"name"`
	InstanceType         string            `json:"instance_type"`
	State                string            `json:"state"`
	Region               string            `json:"region"`
	AvailabilityZone     string            `json:"availability_zone"`
	PublicIP             string            `json:"public_ip,omitempty"`
	PrivateIP            string            `json:"private_ip,omitempty"`
	LaunchTime           time.Time         `json:"launch_time"`
	TTL                  string            `json:"ttl,omitempty"`
	TTLRemainingSeconds  int               `json:"ttl_remaining_seconds,omitempty"`
	IdleTimeout          string            `json:"idle_timeout,omitempty"`
	DNSName              string            `json:"dns_name,omitempty"`
	SpotInstance         bool              `json:"spot_instance"`
	KeyName              string            `json:"key_name,omitempty"`
	Tags                 map[string]string `json:"tags"`
}

// UserProfile represents user account information
type UserProfile struct {
	UserID         string    `json:"user_id"`
	AWSAccountID   string    `json:"aws_account_id"`
	AccountBase36  string    `json:"account_base36"`
	Email          string    `json:"email,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	LastAccess     time.Time `json:"last_access"`
}

// DynamoDB structures

// UserAccountRecord represents a record in the spawn-user-accounts DynamoDB table
type UserAccountRecord struct {
	UserID        string `dynamodbav:"user_id"`
	AWSAccountID  string `dynamodbav:"aws_account_id"`
	AccountBase36 string `dynamodbav:"account_base36"`
	Email         string `dynamodbav:"email,omitempty"`
	CreatedAt     string `dynamodbav:"created_at"`
	LastAccess    string `dynamodbav:"last_access"`
}
