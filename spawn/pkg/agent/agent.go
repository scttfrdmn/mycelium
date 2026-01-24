package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/scttfrdmn/mycelium/spawn/pkg/dns"
)

type Agent struct {
	instanceID       string
	region           string
	accountID        string
	publicIP         string
	dnsName          string
	jobArrayID       string // Empty if not part of job array
	jobArrayName     string // Empty if not part of job array
	dnsClient        *dns.Client
	ec2Client        *ec2.Client
	imdsClient       *imds.Client
	config           AgentConfig
	startTime        time.Time
	lastActivityTime time.Time
}

type AgentConfig struct {
	TTL             time.Duration
	IdleTimeout     time.Duration
	HibernateOnIdle bool
	CostLimit       float64
	IdleCPUPercent  float64

	// Completion signal settings
	OnComplete      string        // Action: terminate, stop, hibernate
	CompletionFile  string        // File path to watch
	CompletionDelay time.Duration // Grace period before action
}

func NewAgent(ctx context.Context) (*Agent, error) {
	// Get instance metadata
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	imdsClient := imds.NewFromConfig(cfg)

	// Get instance identity document
	idDoc, err := imdsClient.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get instance identity: %w", err)
	}

	instanceID := idDoc.InstanceID
	region := idDoc.Region
	accountID := idDoc.AccountID

	// Get public IP from metadata
	publicIPResult, err := imdsClient.GetMetadata(ctx, &imds.GetMetadataInput{
		Path: "public-ipv4",
	})
	var publicIP string
	if err == nil {
		ipBytes, _ := io.ReadAll(publicIPResult.Content)
		publicIP = strings.TrimSpace(string(ipBytes))
	}

	// Update config with region
	cfg.Region = region
	ec2Client := ec2.NewFromConfig(cfg)

	// Get instance tags to read configuration
	agentConfig, dnsName, jobArrayID, jobArrayName, err := loadConfigFromTags(ctx, ec2Client, instanceID)
	if err != nil {
		log.Printf("Warning: Could not load config from tags: %v", err)
		agentConfig = AgentConfig{
			IdleCPUPercent: 5.0,
		}
	}

	agent := &Agent{
		instanceID:       instanceID,
		region:           region,
		accountID:        accountID,
		publicIP:         publicIP,
		dnsName:          dnsName,
		jobArrayID:       jobArrayID,
		jobArrayName:     jobArrayName,
		ec2Client:        ec2Client,
		imdsClient:       imdsClient,
		config:           agentConfig,
		startTime:        time.Now(),
		lastActivityTime: time.Now(),
	}

	log.Printf("Agent initialized for instance %s in %s (account: %s)", instanceID, region, accountID)
	log.Printf("Config: TTL=%v, IdleTimeout=%v, Hibernate=%v",
		agentConfig.TTL, agentConfig.IdleTimeout, agentConfig.HibernateOnIdle)

	// Initialize DNS client and register if DNS name is configured
	if dnsName != "" && publicIP != "" {
		dnsClient, err := dns.NewClient(ctx, "", "") // Use defaults
		if err != nil {
			log.Printf("Warning: Failed to create DNS client: %v", err)
		} else {
			agent.dnsClient = dnsClient

			// Register DNS (use job array method if part of a job array)
			if jobArrayID != "" && jobArrayName != "" {
				log.Printf("Registering job array DNS: %s -> %s (array: %s)", dnsName, publicIP, jobArrayName)
				resp, err := dnsClient.RegisterJobArrayDNS(ctx, dnsName, publicIP, jobArrayID, jobArrayName)
				if err != nil {
					log.Printf("Warning: Failed to register job array DNS: %v", err)
				} else {
					fqdn := dns.GetFullDNSName(dnsName, accountID, "spore.host")
					log.Printf("✓ Job array DNS registered: %s -> %s (change: %s)", fqdn, publicIP, resp.ChangeID)
					if resp.Message != "" {
						log.Printf("  %s", resp.Message)
					}
				}
			} else {
				log.Printf("Registering DNS: %s -> %s", dnsName, publicIP)
				resp, err := dnsClient.RegisterDNS(ctx, dnsName, publicIP)
				if err != nil {
					log.Printf("Warning: Failed to register DNS: %v", err)
				} else {
					fqdn := dns.GetFullDNSName(dnsName, accountID, "spore.host")
					log.Printf("✓ DNS registered: %s -> %s (change: %s)", fqdn, publicIP, resp.ChangeID)
				}
			}
		}
	} else if dnsName != "" {
		log.Printf("Warning: DNS name configured (%s) but no public IP available", dnsName)
	}

	// Load job array peer information if part of a job array
	err = agent.loadJobArrayPeers(ctx)
	if err != nil {
		log.Printf("Warning: Failed to load job array peers: %v", err)
		// Non-fatal: continue without peer information
	}

	return agent, nil
}

func loadConfigFromTags(ctx context.Context, client *ec2.Client, instanceID string) (AgentConfig, string, string, string, error) {
	output, err := client.DescribeTags(ctx, &ec2.DescribeTagsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("resource-id"),
				Values: []string{instanceID},
			},
		},
	})
	if err != nil {
		return AgentConfig{}, "", "", "", err
	}

	config := AgentConfig{
		IdleCPUPercent: 5.0, // Default
	}
	var dnsName, jobArrayID, jobArrayName string

	for _, tag := range output.Tags {
		if tag.Key == nil || tag.Value == nil {
			continue
		}

		switch *tag.Key {
		case "spawn:ttl":
			if duration, err := time.ParseDuration(*tag.Value); err == nil {
				config.TTL = duration
			}
		case "spawn:idle-timeout":
			if duration, err := time.ParseDuration(*tag.Value); err == nil {
				config.IdleTimeout = duration
			}
		case "spawn:hibernate-on-idle":
			config.HibernateOnIdle = *tag.Value == "true"
		case "spawn:cost-limit":
			if limit, err := strconv.ParseFloat(*tag.Value, 64); err == nil {
				config.CostLimit = limit
			}
		case "spawn:idle-cpu":
			if cpu, err := strconv.ParseFloat(*tag.Value, 64); err == nil {
				config.IdleCPUPercent = cpu
			}
		case "spawn:dns-name":
			dnsName = *tag.Value
		case "spawn:on-complete":
			config.OnComplete = *tag.Value
		case "spawn:completion-file":
			config.CompletionFile = *tag.Value
		case "spawn:completion-delay":
			if duration, err := time.ParseDuration(*tag.Value); err == nil {
				config.CompletionDelay = duration
			}
		case "spawn:job-array-id":
			jobArrayID = *tag.Value
		case "spawn:job-array-name":
			jobArrayName = *tag.Value
		}
	}

	// Set default completion file if on-complete is set but file isn't specified
	if config.OnComplete != "" && config.CompletionFile == "" {
		config.CompletionFile = "/tmp/SPAWN_COMPLETE"
	}

	// Set default completion delay if on-complete is set but delay isn't specified
	if config.OnComplete != "" && config.CompletionDelay == 0 {
		config.CompletionDelay = 30 * time.Second
	}

	return config, dnsName, jobArrayID, jobArrayName, nil
}

// PeerInfo represents information about a peer instance in a job array
type PeerInfo struct {
	Index      int    `json:"index"`
	InstanceID string `json:"instance_id"`
	IP         string `json:"ip"`
	DNS        string `json:"dns"`
}

// loadJobArrayPeers discovers peer instances and writes peer information to /etc/spawn/job-array-peers.json
func (a *Agent) loadJobArrayPeers(ctx context.Context) error {
	// Get this instance's job array ID tag
	output, err := a.ec2Client.DescribeTags(ctx, &ec2.DescribeTagsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("resource-id"),
				Values: []string{a.instanceID},
			},
			{
				Name:   aws.String("key"),
				Values: []string{"spawn:job-array-id"},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to describe tags: %w", err)
	}

	// If no job-array-id tag, not part of a job array
	if len(output.Tags) == 0 {
		return nil
	}

	jobArrayID := *output.Tags[0].Value
	log.Printf("Instance is part of job array: %s", jobArrayID)

	// Query for all instances with the same job-array-id
	instances, err := a.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:spawn:job-array-id"),
				Values: []string{jobArrayID},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"pending", "running"},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to query job array instances: %w", err)
	}

	// Build peer list
	var peers []PeerInfo
	accountBase36 := intToBase36(a.accountID)

	for _, reservation := range instances.Reservations {
		for _, instance := range reservation.Instances {
			// Extract job array index from tags
			var index int
			var name string
			for _, tag := range instance.Tags {
				if *tag.Key == "spawn:job-array-index" {
					index, _ = strconv.Atoi(*tag.Value)
				}
				if *tag.Key == "Name" {
					name = *tag.Value
				}
			}

			publicIP := ""
			if instance.PublicIpAddress != nil {
				publicIP = *instance.PublicIpAddress
			}

			// Generate DNS name: {name}.{account-base36}.spore.host
			dnsName := fmt.Sprintf("%s.%s.spore.host", name, accountBase36)

			peer := PeerInfo{
				Index:      index,
				InstanceID: *instance.InstanceId,
				IP:         publicIP,
				DNS:        dnsName,
			}
			peers = append(peers, peer)
		}
	}

	// Sort by index
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].Index < peers[j].Index
	})

	// Create /etc/spawn directory if it doesn't exist
	err = os.MkdirAll("/etc/spawn", 0755)
	if err != nil {
		return fmt.Errorf("failed to create /etc/spawn directory: %w", err)
	}

	// Marshal to JSON and write to file
	peersJSON, err := json.MarshalIndent(peers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal peers: %w", err)
	}

	peersFile := "/etc/spawn/job-array-peers.json"
	err = os.WriteFile(peersFile, peersJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write peers file: %w", err)
	}

	log.Printf("✓ Job array peer information written to %s (%d peers)", peersFile, len(peers))

	return nil
}

// intToBase36 converts an AWS account ID to base36
func intToBase36(accountID string) string {
	num, err := strconv.ParseUint(accountID, 10, 64)
	if err != nil {
		return accountID
	}
	return strconv.FormatUint(num, 36)
}

func (a *Agent) Monitor(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Printf("Monitoring started")

	for {
		select {
		case <-ctx.Done():
			log.Printf("Context cancelled, stopping monitor")
			return

		case <-ticker.C:
			a.checkAndAct(ctx)
		}
	}
}

func (a *Agent) checkAndAct(ctx context.Context) {
	// 0. Check for Spot interruption (HIGHEST PRIORITY)
	if a.checkSpotInterruption(ctx) {
		// Spot interruption detected - handled in checkSpotInterruption
		return
	}

	// 1. Check for completion signal (HIGH PRIORITY)
	if a.config.OnComplete != "" {
		if a.checkCompletion(ctx) {
			// Completion signal detected - handled in checkCompletion
			return
		}
	}

	// 2. Check TTL
	if a.config.TTL > 0 {
		uptime := time.Since(a.startTime)
		remaining := a.config.TTL - uptime

		if remaining <= 0 {
			log.Printf("TTL expired (limit: %v, uptime: %v)", a.config.TTL, uptime)
			a.terminate(ctx, "TTL expired")
			return
		}

		// Warn at 5 minutes
		if remaining > 0 && remaining <= 5*time.Minute {
			a.warnUsers(fmt.Sprintf("⚠️  TERMINATING IN %v (TTL limit)", remaining.Round(time.Minute)))
		}
	}

	// 3. Check idle
	if a.config.IdleTimeout > 0 {
		idle := a.isIdle()
		if idle {
			idleTime := time.Since(a.lastActivityTime)

			if idleTime >= a.config.IdleTimeout {
				log.Printf("Idle timeout reached (%v)", idleTime)

				if a.config.HibernateOnIdle {
					a.hibernate(ctx)
				} else {
					a.terminate(ctx, "Idle timeout")
				}
				return
			}

			// Warn at 5 minutes before idle timeout
			remaining := a.config.IdleTimeout - idleTime
			if remaining > 0 && remaining <= 5*time.Minute {
				a.warnUsers(fmt.Sprintf("⚠️  IDLE for %v, will terminate in %v",
					idleTime.Round(time.Minute), remaining.Round(time.Minute)))
			}
		} else {
			// Activity detected, reset timer
			a.lastActivityTime = time.Now()
		}
	}
}

func (a *Agent) isIdle() bool {
	// Check CPU usage
	cpuUsage := a.getCPUUsage()
	if cpuUsage >= a.config.IdleCPUPercent {
		log.Printf("Not idle: CPU usage %.2f%% >= %.2f%%", cpuUsage, a.config.IdleCPUPercent)
		return false
	}

	// Check network traffic
	networkBytes := a.getNetworkBytes()
	if networkBytes > 10000 { // 10KB/min threshold
		log.Printf("Not idle: Network traffic %d bytes", networkBytes)
		return false
	}

	// Check disk I/O
	diskIO := a.getDiskIO()
	if diskIO > 100000 { // 100KB/min threshold
		log.Printf("Not idle: Disk I/O %d bytes", diskIO)
		return false
	}

	// Check GPU utilization
	gpuUtilization := a.getGPUUtilization()
	if gpuUtilization > 5 { // 5% GPU usage threshold
		log.Printf("Not idle: GPU utilization %.2f%%", gpuUtilization)
		return false
	}

	// Check for active terminals
	if a.hasActiveTerminals() {
		log.Printf("Not idle: Active terminals present")
		return false
	}

	// Check for logged-in users
	if a.hasLoggedInUsers() {
		log.Printf("Not idle: Users logged in")
		return false
	}

	// Check for recent user activity
	if a.hasRecentUserActivity() {
		log.Printf("Not idle: Recent user activity detected")
		return false
	}

	log.Printf("System is idle (CPU: %.2f%%, Network: %d bytes, Disk: %d bytes, GPU: %.2f%%)",
		cpuUsage, networkBytes, diskIO, gpuUtilization)
	return true
}

func (a *Agent) getCPUUsage() float64 {
	// Read /proc/stat
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 100.0 // Assume active if can't read
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 100.0
	}

	// Parse first line: cpu  user nice system idle ...
	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 100.0
	}

	// Simple idle check - if idle column is very high, system is idle
	idle, _ := strconv.ParseFloat(fields[4], 64)
	total := 0.0
	for i := 1; i < len(fields); i++ {
		val, _ := strconv.ParseFloat(fields[i], 64)
		total += val
	}

	if total == 0 {
		return 0
	}

	// Return usage percentage
	return 100.0 - (idle/total)*100.0
}

func (a *Agent) getNetworkBytes() int64 {
	// Read /proc/net/dev
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 1000000 // Assume active if can't read
	}

	lines := strings.Split(string(data), "\n")
	var totalBytes int64

	for _, line := range lines {
		if strings.Contains(line, "eth0") || strings.Contains(line, "ens") {
			fields := strings.Fields(line)
			if len(fields) >= 10 {
				// RX bytes + TX bytes
				rx, _ := strconv.ParseInt(fields[1], 10, 64)
				tx, _ := strconv.ParseInt(fields[9], 10, 64)
				totalBytes += rx + tx
			}
		}
	}

	return totalBytes
}

func (a *Agent) getDiskIO() int64 {
	// Read /proc/diskstats
	// Format: major minor name reads ... sectors_read ... writes ... sectors_written ...
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return 0 // Assume no activity if can't read
	}

	lines := strings.Split(string(data), "\n")
	var totalSectors int64

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}

		// Check for main block devices (skip partitions)
		deviceName := fields[2]
		if strings.HasPrefix(deviceName, "xvd") || strings.HasPrefix(deviceName, "nvme") ||
			strings.HasPrefix(deviceName, "sd") || strings.HasPrefix(deviceName, "vd") {
			// Skip partition numbers (xvda1, nvme0n1p1, etc.)
			if len(deviceName) > 4 && deviceName[len(deviceName)-1] >= '0' && deviceName[len(deviceName)-1] <= '9' {
				// Check if it's a partition (has digit at end)
				continue
			}

			// Fields: 0=major 1=minor 2=name 3=reads 4=reads_merged 5=sectors_read
			// 6=time_reading 7=writes 8=writes_merged 9=sectors_written 10=time_writing
			sectorsRead, _ := strconv.ParseInt(fields[5], 10, 64)
			sectorsWritten, _ := strconv.ParseInt(fields[9], 10, 64)
			totalSectors += sectorsRead + sectorsWritten
		}
	}

	// Convert sectors to bytes (typically 512 bytes per sector)
	return totalSectors * 512
}

func (a *Agent) getGPUUtilization() float64 {
	// Check if nvidia-smi is available
	_, err := exec.LookPath("nvidia-smi")
	if err != nil {
		// No GPU or nvidia-smi not installed
		return 0
	}

	// Query GPU utilization
	// nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits
	cmd := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Parse output (can have multiple GPUs, one per line)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var maxUtilization float64

	for _, line := range lines {
		utilization, err := strconv.ParseFloat(strings.TrimSpace(line), 64)
		if err == nil && utilization > maxUtilization {
			maxUtilization = utilization
		}
	}

	return maxUtilization
}

func (a *Agent) hasLoggedInUsers() bool {
	// Use 'who' command to check for logged-in users
	cmd := exec.Command("who")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// If output is not empty, users are logged in
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			return true
		}
	}

	return false
}

func (a *Agent) hasRecentUserActivity() bool {
	// Check for recent activity in wtmp (last 5 minutes)
	// Use 'last -s -5min' to check recent logins
	cmd := exec.Command("last", "-s", "-5min", "-w")
	output, err := cmd.Output()
	if err != nil {
		// If 'last' fails, check /var/log/wtmp modification time
		fileInfo, err := os.Stat("/var/log/wtmp")
		if err != nil {
			return false
		}
		// If modified in last 5 minutes, there was activity
		return time.Since(fileInfo.ModTime()) < 5*time.Minute
	}

	// Parse output - if there are login entries, there was recent activity
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	// Skip header lines and empty lines
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "wtmp") && !strings.HasPrefix(line, "reboot") {
			// Check if it's a user login line (not system events)
			if !strings.Contains(line, "system boot") && !strings.Contains(line, "down") {
				return true
			}
		}
	}

	return false
}

func (a *Agent) hasActiveTerminals() bool {
	// Check for active pseudo-terminals in /dev/pts/
	// This detects interactive SSH sessions or other terminal sessions
	entries, err := os.ReadDir("/dev/pts")
	if err != nil {
		return false
	}

	// Count active PTYs (exclude ptmx which is the multiplexer)
	activeCount := 0
	for _, entry := range entries {
		name := entry.Name()
		// Skip ptmx (the master pseudo-terminal multiplexer)
		if name == "ptmx" {
			continue
		}

		// Check if it's a number (active PTY)
		if _, err := strconv.Atoi(name); err == nil {
			activeCount++
		}
	}

	// If there are active PTYs, terminals are present
	return activeCount > 0
}

func (a *Agent) checkSpotInterruption(ctx context.Context) bool {
	// Check if this is a Spot instance
	if !a.isSpotInstance() {
		return false
	}

	// Query the Spot instance action metadata
	// http://169.254.169.254/latest/meta-data/spot/instance-action
	// Returns 404 if no interruption, or JSON like:
	// {"action": "terminate", "time": "2023-11-30T12:34:56Z"}

	result, err := a.imdsClient.GetMetadata(ctx, &imds.GetMetadataInput{
		Path: "spot/instance-action",
	})

	if err != nil {
		// 404 means no interruption notice
		if strings.Contains(err.Error(), "404") {
			return false
		}
		// Other errors - log but don't treat as interruption
		log.Printf("Error checking Spot interruption: %v", err)
		return false
	}

	// Parse the response
	body, err := io.ReadAll(result.Content)
	if err != nil {
		log.Printf("Error reading Spot interruption response: %v", err)
		return false
	}

	// Log the raw interruption notice
	log.Printf("🚨 SPOT INTERRUPTION DETECTED: %s", string(body))

	// Parse JSON to get details
	var action struct {
		Action string `json:"action"`
		Time   string `json:"time"`
	}

	if err := json.Unmarshal(body, &action); err != nil {
		log.Printf("Error parsing Spot interruption JSON: %v", err)
	}

	// Clean up DNS immediately to avoid stale records
	log.Printf("Spot interruption: Running cleanup tasks")
	cleanupCtx := context.Background()
	a.Cleanup(cleanupCtx)

	// Alert users immediately
	message := fmt.Sprintf("🚨 SPOT INTERRUPTION WARNING! 🚨\n"+
		"AWS will %s this instance at %s\n"+
		"You have ~2 minutes to save your work!\n"+
		"SAVE ALL FILES NOW!", action.Action, action.Time)

	a.warnUsers(message)

	// Send notifications (if configured)
	a.sendSpotInterruptionNotification(action.Action, action.Time)

	// Log for posterity
	log.Printf("Spot interruption: action=%s, time=%s", action.Action, action.Time)

	// Continue monitoring for remaining time
	// (don't return immediately, let other checks continue)
	return false // Return false to allow normal monitoring to continue
}

func (a *Agent) isSpotInstance() bool {
	// Check if we're running on a Spot instance via instance lifecycle metadata
	result, err := a.imdsClient.GetMetadata(context.Background(), &imds.GetMetadataInput{
		Path: "instance-life-cycle",
	})
	if err != nil {
		return false
	}

	body, err := io.ReadAll(result.Content)
	if err != nil {
		return false
	}

	lifecycle := strings.TrimSpace(string(body))
	return lifecycle == "spot"
}

func (a *Agent) sendSpotInterruptionNotification(action, interruptTime string) {
	// Log to spored logs (always)
	log.Printf("📢 NOTIFICATION: Spot interruption detected - action=%s time=%s", action, interruptTime)

	// Write to a file that can be picked up by external systems
	notificationFile := "/tmp/spawn-spot-interruption.json"
	notification := fmt.Sprintf(`{
  "event": "spot-interruption",
  "instance_id": "%s",
  "action": "%s",
  "time": "%s",
  "detected_at": "%s"
}`, a.instanceID, action, interruptTime, time.Now().UTC().Format(time.RFC3339))

	if err := os.WriteFile(notificationFile, []byte(notification), 0644); err != nil {
		log.Printf("Failed to write notification file: %v", err)
	}

	// Future enhancement: Support webhooks, email, SNS, etc.
	// For now, the notification file can be picked up by external monitoring
}

func (a *Agent) warnUsers(message string) {
	// Write to all logged-in terminals
	cmd := exec.Command("wall", message)
	cmd.Run()

	// Also write to a warning file
	os.WriteFile("/tmp/SPAWN_WARNING", []byte(message+"\n"), 0644)

	log.Printf("Warning sent to users: %s", message)
}

func (a *Agent) checkCompletion(ctx context.Context) bool {
	// Check if completion file exists
	if _, err := os.Stat(a.config.CompletionFile); err == nil {
		log.Printf("Completion signal detected: file %s exists", a.config.CompletionFile)

		// Read completion file for metadata (optional)
		content, err := os.ReadFile(a.config.CompletionFile)
		if err == nil && len(content) > 0 {
			log.Printf("Completion metadata: %s", strings.TrimSpace(string(content)))
		}

		// Warn users with grace period
		delay := a.config.CompletionDelay
		a.warnUsers(fmt.Sprintf("✓ Workload complete - %s in %v", a.config.OnComplete, delay))

		log.Printf("Grace period: waiting %v before action", delay)
		time.Sleep(delay)

		// Execute action based on configuration
		switch strings.ToLower(a.config.OnComplete) {
		case "terminate":
			a.terminate(ctx, "Completion signal received")
		case "stop":
			a.stop(ctx, "Completion signal received")
		case "hibernate":
			a.hibernate(ctx)
		default:
			log.Printf("Unknown on-complete action: %s (doing nothing)", a.config.OnComplete)
			return false
		}

		return true
	}

	return false
}

func (a *Agent) stop(ctx context.Context, reason string) {
	log.Printf("Stopping instance %s (reason: %s)", a.instanceID, reason)

	// Clean up DNS before stopping
	a.Cleanup(ctx)

	a.warnUsers(fmt.Sprintf("🛑 STOPPING NOW - %s", reason))

	// Wait a moment for users to see warning
	time.Sleep(5 * time.Second)

	_, err := a.ec2Client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{a.instanceID},
	})

	if err != nil {
		log.Printf("Failed to stop instance: %v", err)
	} else {
		log.Printf("Stop request sent")
	}
}

func (a *Agent) hibernate(ctx context.Context) {
	log.Printf("Hibernating instance %s", a.instanceID)

	// Clean up DNS before hibernating
	a.Cleanup(ctx)

	a.warnUsers("💤 HIBERNATING NOW - Instance will pause, resume later")

	// Wait a moment for users to see warning
	time.Sleep(5 * time.Second)

	_, err := a.ec2Client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{a.instanceID},
		Hibernate:   aws.Bool(true),
	})

	if err != nil {
		log.Printf("Failed to hibernate: %v", err)
		// Fall back to regular stop
		a.ec2Client.StopInstances(ctx, &ec2.StopInstancesInput{
			InstanceIds: []string{a.instanceID},
		})
	}

	log.Printf("Hibernate request sent")
}

// Cleanup performs cleanup tasks before shutdown (DNS deregistration, etc.)
func (a *Agent) Cleanup(ctx context.Context) {
	log.Printf("Running cleanup tasks...")

	// Clean up DNS
	if a.dnsClient != nil && a.dnsName != "" && a.publicIP != "" {
		cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// Use job array DNS deletion if part of a job array
		if a.jobArrayID != "" && a.jobArrayName != "" {
			log.Printf("Deleting job array DNS record: %s (array: %s)", a.dnsName, a.jobArrayName)
			resp, err := a.dnsClient.DeleteJobArrayDNS(cleanupCtx, a.dnsName, a.publicIP, a.jobArrayID, a.jobArrayName)
			if err != nil {
				log.Printf("Warning: Failed to delete job array DNS: %v", err)
			} else {
				fqdn := dns.GetFullDNSName(a.dnsName, a.accountID, "spore.host")
				log.Printf("✓ Job array DNS deleted: %s", fqdn)
				if resp.Message != "" {
					log.Printf("  %s", resp.Message)
				}
			}
		} else {
			log.Printf("Deleting DNS record: %s", a.dnsName)
			_, err := a.dnsClient.DeleteDNS(cleanupCtx, a.dnsName, a.publicIP)
			if err != nil {
				log.Printf("Warning: Failed to delete DNS: %v", err)
			} else {
				fqdn := dns.GetFullDNSName(a.dnsName, a.accountID, "spore.host")
				log.Printf("✓ DNS deleted: %s", fqdn)
			}
		}
	}

	log.Printf("Cleanup complete")
}

func (a *Agent) terminate(ctx context.Context, reason string) {
	log.Printf("Terminating instance %s (reason: %s)", a.instanceID, reason)

	// Clean up DNS before terminating
	a.Cleanup(ctx)

	a.warnUsers(fmt.Sprintf("🔴 TERMINATING NOW - Reason: %s", reason))

	// Wait a moment for users to see warning
	time.Sleep(5 * time.Second)

	_, err := a.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{a.instanceID},
	})

	if err != nil {
		log.Printf("Failed to terminate: %v", err)
	} else {
		log.Printf("Terminate request sent")
	}
}

// Reload re-reads configuration from EC2 tags without restarting the daemon
func (a *Agent) Reload(ctx context.Context) error {
	log.Printf("Reloading configuration from tags...")

	// Re-read tags
	newConfig, newDNSName, newJobArrayID, newJobArrayName, err := loadConfigFromTags(ctx, a.ec2Client, a.instanceID)
	if err != nil {
		return fmt.Errorf("failed to reload config from tags: %w", err)
	}

	// Update job array fields (they shouldn't change after launch, but update anyway)
	a.jobArrayID = newJobArrayID
	a.jobArrayName = newJobArrayName

	// Log changes
	if newConfig.TTL != a.config.TTL {
		log.Printf("TTL changed: %v → %v", a.config.TTL, newConfig.TTL)
	}
	if newConfig.IdleTimeout != a.config.IdleTimeout {
		log.Printf("Idle timeout changed: %v → %v", a.config.IdleTimeout, newConfig.IdleTimeout)
	}
	if newConfig.OnComplete != a.config.OnComplete {
		log.Printf("On-complete changed: %s → %s", a.config.OnComplete, newConfig.OnComplete)
	}
	if newConfig.HibernateOnIdle != a.config.HibernateOnIdle {
		log.Printf("Hibernate-on-idle changed: %v → %v", a.config.HibernateOnIdle, newConfig.HibernateOnIdle)
	}

	// Update config (but keep startTime - TTL is absolute)
	a.config = newConfig
	a.dnsName = newDNSName

	log.Printf("Configuration reloaded successfully")
	log.Printf("New config: TTL=%v, IdleTimeout=%v, OnComplete=%s, Hibernate=%v",
		newConfig.TTL, newConfig.IdleTimeout, newConfig.OnComplete, newConfig.HibernateOnIdle)

	return nil
}

// Public getter methods for status reporting

func (a *Agent) GetConfig() AgentConfig {
	return a.config
}

func (a *Agent) GetInstanceInfo() (string, string, string) {
	return a.instanceID, a.region, a.accountID
}

func (a *Agent) GetUptime() time.Duration {
	return time.Since(a.startTime)
}

func (a *Agent) GetCPUUsage() float64 {
	return a.getCPUUsage()
}

func (a *Agent) GetNetworkBytes() int64 {
	return a.getNetworkBytes()
}

func (a *Agent) IsIdle() bool {
	return a.isIdle()
}

func (a *Agent) GetLastActivityTime() time.Time {
	return a.lastActivityTime
}

// UX detection methods
func (a *Agent) HasActiveTerminals() bool {
	return a.hasActiveTerminals()
}

func (a *Agent) HasLoggedInUsers() bool {
	return a.hasLoggedInUsers()
}

func (a *Agent) HasRecentUserActivity() bool {
	return a.hasRecentUserActivity()
}
