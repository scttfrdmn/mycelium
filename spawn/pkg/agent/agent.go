package agent

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type Agent struct {
	instanceID      string
	region          string
	ec2Client       *ec2.Client
	imdsClient      *imds.Client
	config          AgentConfig
	startTime       time.Time
	lastActivityTime time.Time
}

type AgentConfig struct {
	TTL             time.Duration
	IdleTimeout     time.Duration
	HibernateOnIdle bool
	CostLimit       float64
	IdleCPUPercent  float64
}

func NewAgent(ctx context.Context) (*Agent, error) {
	// Get instance metadata
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	imdsClient := imds.NewFromConfig(cfg)

	// Get instance ID
	idDoc, err := imdsClient.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get instance identity: %w", err)
	}

	instanceID := idDoc.InstanceID
	region := idDoc.Region

	// Update config with region
	cfg.Region = region
	ec2Client := ec2.NewFromConfig(cfg)

	// Get instance tags to read configuration
	agentConfig, err := loadConfigFromTags(ctx, ec2Client, instanceID)
	if err != nil {
		log.Printf("Warning: Could not load config from tags: %v", err)
		agentConfig = AgentConfig{
			IdleCPUPercent: 5.0,
		}
	}

	agent := &Agent{
		instanceID:       instanceID,
		region:           region,
		ec2Client:        ec2Client,
		imdsClient:       imdsClient,
		config:           agentConfig,
		startTime:        time.Now(),
		lastActivityTime: time.Now(),
	}

	log.Printf("Agent initialized for instance %s in %s", instanceID, region)
	log.Printf("Config: TTL=%v, IdleTimeout=%v, Hibernate=%v",
		agentConfig.TTL, agentConfig.IdleTimeout, agentConfig.HibernateOnIdle)

	return agent, nil
}

func loadConfigFromTags(ctx context.Context, client *ec2.Client, instanceID string) (AgentConfig, error) {
	output, err := client.DescribeTags(ctx, &ec2.DescribeTagsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("resource-id"),
				Values: []string{instanceID},
			},
		},
	})
	if err != nil {
		return AgentConfig{}, err
	}

	config := AgentConfig{
		IdleCPUPercent: 5.0, // Default
	}

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
		}
	}

	return config, nil
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
	// 1. Check TTL
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

	// 2. Check idle
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
		return false
	}

	// Check network traffic
	networkBytes := a.getNetworkBytes()
	if networkBytes > 10000 { // 10KB/min threshold
		return false
	}

	return true
}

func (a *Agent) getCPUUsage() float64 {
	// Read /proc/stat
	data, err := ioutil.ReadFile("/proc/stat")
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
	data, err := ioutil.ReadFile("/proc/net/dev")
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

func (a *Agent) warnUsers(message string) {
	// Write to all logged-in terminals
	cmd := exec.Command("wall", message)
	cmd.Run()

	// Also write to a warning file
	os.WriteFile("/tmp/SPAWN_WARNING", []byte(message+"\n"), 0644)

	log.Printf("Warning sent to users: %s", message)
}

func (a *Agent) hibernate(ctx context.Context) {
	log.Printf("Hibernating instance %s", a.instanceID)

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

func (a *Agent) terminate(ctx context.Context, reason string) {
	log.Printf("Terminating instance %s (reason: %s)", a.instanceID, reason)

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
