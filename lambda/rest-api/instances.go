package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	spawnclient "github.com/spore-host/spawn/pkg/aws"
	spawnconfig "github.com/spore-host/spawn/pkg/config"
)

// projectTag is the EC2 tag that scopes an instance to the API-key principal's
// project. The hosted API is single-account: every launch runs as the same
// Lambda role, so identity tags (spawn:iam-user, spawn:account-id) are identical
// across tenants and can't isolate anything. The principal's project is the only
// real tenant discriminator, so we stamp it on launch and gate every read/action
// on it (spore-host#369).
const projectTag = "spawn:project"

// ownsInstance reports whether the principal may see/act on an instance. It is
// fail-closed: a principal with an empty project, or an instance with no/blank
// project tag, never matches — so a leaked key with no project can't reach
// pre-existing untagged instances, and an empty project can't act as a wildcard.
func ownsInstance(p *Principal, tags map[string]string) bool {
	if p == nil || p.Project == "" {
		return false
	}
	return tags[projectTag] == p.Project
}

func handleListInstances(ctx context.Context, cfg aws.Config, req events.APIGatewayV2HTTPRequest, p *Principal) (events.APIGatewayV2HTTPResponse, error) {
	region := req.QueryStringParameters["region"]
	state := req.QueryStringParameters["state"]
	if state == "" {
		state = "running"
	}

	client := spawnclient.NewClientFromConfig(cfg)
	instances, err := client.ListInstances(ctx, region, state)
	if err != nil {
		return errResp(http.StatusInternalServerError, fmt.Sprintf("list instances: %v", err)), nil
	}

	type instanceOut struct {
		InstanceID       string            `json:"instance_id"`
		Name             string            `json:"name"`
		InstanceType     string            `json:"instance_type"`
		State            string            `json:"state"`
		Region           string            `json:"region"`
		AvailabilityZone string            `json:"availability_zone,omitempty"`
		PublicIP         string            `json:"public_ip,omitempty"`
		DNS              string            `json:"dns,omitempty"`
		LaunchTime       time.Time         `json:"launch_time"`
		TTL              string            `json:"ttl,omitempty"`
		IdleTimeout      string            `json:"idle_timeout,omitempty"`
		Tags             map[string]string `json:"tags,omitempty"`
	}

	out := make([]instanceOut, 0, len(instances))
	for _, inst := range instances {
		// Tenant isolation: only surface instances belonging to this principal's
		// project (spore-host#369).
		if !ownsInstance(p, inst.Tags) {
			continue
		}
		dns := inst.Tags["spawn:dns-name"]
		out = append(out, instanceOut{
			InstanceID:       inst.InstanceID,
			Name:             inst.Name,
			InstanceType:     inst.InstanceType,
			State:            inst.State,
			Region:           inst.Region,
			AvailabilityZone: inst.AvailabilityZone,
			PublicIP:         inst.PublicIP,
			DNS:              dns,
			LaunchTime:       inst.LaunchTime,
			TTL:              inst.TTL,
			IdleTimeout:      inst.IdleTimeout,
		})
	}
	return jsonResp(http.StatusOK, map[string]any{"instances": out, "count": len(out)}), nil
}

// Lifecycle bounds for the hosted REST API. Unlike the CLI (which prompts and
// defaults to a 1h idle timeout in cmd/launch.go), the API calls the spawn
// client directly — so the safety net must live here. Without it, an empty-TTL
// launch yields an instance with NO deadline and NO reaper tag that runs until
// manually killed (spore-host#371).
const (
	// defaultIdleTimeout is applied when a launch request sets neither TTL nor
	// idle timeout — mirrors the CLI's zombie-prevention default.
	defaultIdleTimeout = "1h"
	// maxTTL is the hard ceiling on any TTL or extend duration through the API.
	maxTTL = 7 * 24 * time.Hour
)

// capDuration parses a Go duration and rejects values that are non-positive or
// exceed maxTTL. Returns the normalized string form on success.
func capDuration(d string) (string, error) {
	parsed, err := time.ParseDuration(d)
	if err != nil {
		return "", fmt.Errorf("invalid duration %q (use Go units like 2h, 24h, 168h)", d)
	}
	if parsed <= 0 {
		return "", fmt.Errorf("duration %q must be positive", d)
	}
	if parsed > maxTTL {
		return "", fmt.Errorf("duration %q exceeds maximum of %s", d, maxTTL)
	}
	return d, nil
}

// LaunchRequest is the JSON body for POST /v1/instances.
// Only InstanceType, Region, and AMI are required; all lifecycle fields are optional.
type LaunchRequest struct {
	Name            string `json:"name"`
	InstanceType    string `json:"instance_type"`
	Region          string `json:"region"`
	AMI             string `json:"ami"`
	KeyName         string `json:"key_name,omitempty"`
	Spot            bool   `json:"spot,omitempty"`
	TTL             string `json:"ttl,omitempty"`
	IdleTimeout     string `json:"idle_timeout,omitempty"`
	OnComplete      string `json:"on_complete,omitempty"`
	PreStop         string `json:"pre_stop,omitempty"`
	CompletionFile  string `json:"completion_file,omitempty"`
	SlackWorkspace  string `json:"slack_workspace,omitempty"`
	ActiveProcesses string `json:"active_processes,omitempty"`
}

func handleLaunch(ctx context.Context, cfg aws.Config, req events.APIGatewayV2HTTPRequest, p *Principal) (events.APIGatewayV2HTTPResponse, error) {
	var body LaunchRequest
	if err := parseJSON(req.Body, &body); err != nil {
		return errResp(http.StatusBadRequest, "invalid JSON body"), nil
	}
	if body.InstanceType == "" || body.Region == "" {
		return errResp(http.StatusBadRequest, "instance_type and region are required"), nil
	}

	// Tenant isolation (spore-host#369): the launched instance must be stamped
	// with the principal's project so later list/get/action can scope to it.
	// Reject a key with no project rather than create an un-isolated instance.
	if p == nil || p.Project == "" {
		return errResp(http.StatusForbidden, "API key has no project; cannot launch"), nil
	}

	// Enforce lifecycle bounds (spore-host#371). Validate any caller-supplied
	// TTL/idle timeout against the hard maximum, and if BOTH are empty inject a
	// default idle timeout so the instance can never become a zombie.
	if body.TTL != "" {
		if _, err := capDuration(body.TTL); err != nil {
			return errResp(http.StatusBadRequest, err.Error()), nil
		}
	}
	if body.IdleTimeout != "" {
		if _, err := capDuration(body.IdleTimeout); err != nil {
			return errResp(http.StatusBadRequest, err.Error()), nil
		}
	}
	if body.TTL == "" && body.IdleTimeout == "" {
		body.IdleTimeout = defaultIdleTimeout
	}

	lc := spawnclient.LaunchConfig{
		Name:               body.Name,
		InstanceType:       body.InstanceType,
		Region:             body.Region,
		AMI:                body.AMI,
		KeyName:            body.KeyName,
		Spot:               body.Spot,
		TTL:                body.TTL,
		IdleTimeout:        body.IdleTimeout,
		OnComplete:         body.OnComplete,
		PreStop:            body.PreStop,
		CompletionFile:     body.CompletionFile,
		SlackWorkspaceID:   body.SlackWorkspace,
		ActiveProcessesRaw: body.ActiveProcesses,
		Tags:               map[string]string{projectTag: p.Project},
	}

	// Inject notification URL for hosted spore.host
	if body.SlackWorkspace != "" && lc.NotifyURL == "" {
		lc.NotifyURL = spawnconfig.GetNotifyURL()
		lc.NotifyCommand = "/spore"
	}

	client := spawnclient.NewClientFromConfig(cfg)
	result, err := client.Launch(ctx, lc)
	if err != nil {
		return errResp(http.StatusInternalServerError, fmt.Sprintf("launch failed: %v", err)), nil
	}

	return jsonResp(http.StatusCreated, map[string]any{
		"instance_id":       result.InstanceID,
		"name":              result.Name,
		"public_ip":         result.PublicIP,
		"private_ip":        result.PrivateIP,
		"availability_zone": result.AvailabilityZone,
		"state":             result.State,
		"key_name":          result.KeyName,
		"region":            body.Region,
	}), nil
}

func handleGetInstance(ctx context.Context, cfg aws.Config, id string, p *Principal) (events.APIGatewayV2HTTPResponse, error) {
	client := spawnclient.NewClientFromConfig(cfg)
	instances, err := client.ListInstances(ctx, "", "")
	if err != nil {
		return errResp(http.StatusInternalServerError, fmt.Sprintf("list: %v", err)), nil
	}
	for _, inst := range instances {
		if inst.InstanceID == id || strings.EqualFold(inst.Name, id) {
			// Tenant isolation: 404 (not 403) for instances outside the
			// principal's project, so we don't leak their existence (#369).
			if !ownsInstance(p, inst.Tags) {
				return errResp(http.StatusNotFound, fmt.Sprintf("instance %q not found", id)), nil
			}
			return jsonResp(http.StatusOK, inst), nil
		}
	}
	return errResp(http.StatusNotFound, fmt.Sprintf("instance %q not found", id)), nil
}

func handleInstanceAction(ctx context.Context, cfg aws.Config, id, action string, req events.APIGatewayV2HTTPRequest, p *Principal) (events.APIGatewayV2HTTPResponse, error) {
	client := spawnclient.NewClientFromConfig(cfg)

	// Resolve instance
	instances, err := client.ListInstances(ctx, "", "")
	if err != nil {
		return errResp(http.StatusInternalServerError, "list instances failed"), nil
	}
	var target *spawnclient.InstanceInfo
	for i := range instances {
		if instances[i].InstanceID == id || strings.EqualFold(instances[i].Name, id) {
			target = &instances[i]
			break
		}
	}
	if target == nil {
		return errResp(http.StatusNotFound, fmt.Sprintf("instance %q not found", id)), nil
	}
	// Tenant isolation: refuse to act on an instance outside the principal's
	// project, masked as 404 so existence isn't leaked (#369).
	if !ownsInstance(p, target.Tags) {
		return errResp(http.StatusNotFound, fmt.Sprintf("instance %q not found", id)), nil
	}

	switch action {
	case "stop":
		if err := client.StopInstance(ctx, target.Region, target.InstanceID, false); err != nil {
			return errResp(http.StatusInternalServerError, fmt.Sprintf("stop failed: %v", err)), nil
		}
		return jsonResp(http.StatusOK, map[string]string{"status": "stopped", "instance_id": target.InstanceID}), nil

	case "hibernate":
		if err := client.StopInstance(ctx, target.Region, target.InstanceID, true); err != nil {
			return errResp(http.StatusInternalServerError, fmt.Sprintf("hibernate failed: %v", err)), nil
		}
		return jsonResp(http.StatusOK, map[string]string{"status": "hibernating", "instance_id": target.InstanceID}), nil

	case "start":
		if err := client.StartInstance(ctx, target.Region, target.InstanceID); err != nil {
			return errResp(http.StatusInternalServerError, fmt.Sprintf("start failed: %v", err)), nil
		}
		return jsonResp(http.StatusOK, map[string]string{"status": "starting", "instance_id": target.InstanceID}), nil

	case "terminate":
		if err := client.Terminate(ctx, target.Region, target.InstanceID); err != nil {
			return errResp(http.StatusInternalServerError, fmt.Sprintf("terminate failed: %v", err)), nil
		}
		return jsonResp(http.StatusOK, map[string]string{"status": "terminating", "instance_id": target.InstanceID}), nil

	case "extend":
		var body struct {
			Duration string `json:"duration"`
		}
		if req.Body != "" {
			if err := parseJSON(req.Body, &body); err != nil || body.Duration == "" {
				return errResp(http.StatusBadRequest, "body must include {\"duration\": \"2h\"}"), nil
			}
		} else {
			body.Duration = req.QueryStringParameters["duration"]
		}
		if body.Duration == "" {
			return errResp(http.StatusBadRequest, "duration required"), nil
		}
		if _, err := capDuration(body.Duration); err != nil {
			return errResp(http.StatusBadRequest, err.Error()), nil
		}
		extendDuration, err := time.ParseDuration(body.Duration)
		if err != nil {
			return errResp(http.StatusBadRequest, fmt.Sprintf("invalid duration: %v", err)), nil
		}
		// spored treats spawn:ttl-deadline (absolute) as authoritative and ignores
		// spawn:ttl. Writing only spawn:ttl is a silent no-op — the instance still
		// dies at its original deadline (same bug class as spore-host-mcp#11). Push
		// the absolute deadline forward and write BOTH tags, mirroring cmd/extend.go.
		var newDeadline time.Time
		if dl, ok := target.Tags["spawn:ttl-deadline"]; ok {
			if parsed, perr := time.Parse(time.RFC3339, dl); perr == nil {
				newDeadline = parsed.Add(extendDuration)
			}
		}
		if newDeadline.IsZero() {
			if cur, cerr := time.ParseDuration(target.TTL); cerr == nil {
				newDeadline = time.Now().Add(cur).Add(extendDuration)
			} else {
				newDeadline = time.Now().Add(extendDuration)
			}
		}
		// Safety floor (2026-06 audit, M-corr): an extend must never produce a
		// deadline earlier than the requested duration from now — otherwise a
		// past/expired existing deadline would reap the instance the moment the
		// caller asked to keep it.
		if floor := time.Now().Add(extendDuration); newDeadline.Before(floor) {
			newDeadline = floor
		}
		if err := client.UpdateInstanceTags(ctx, target.Region, target.InstanceID, map[string]string{
			"spawn:ttl":          body.Duration,
			"spawn:ttl-deadline": newDeadline.UTC().Format(time.RFC3339),
		}); err != nil {
			return errResp(http.StatusInternalServerError, fmt.Sprintf("extend failed: %v", err)), nil
		}
		return jsonResp(http.StatusOK, map[string]string{
			"status":      "extended",
			"ttl":         body.Duration,
			"deadline":    newDeadline.UTC().Format(time.RFC3339),
			"instance_id": target.InstanceID,
		}), nil

	default:
		return errResp(http.StatusBadRequest, fmt.Sprintf("unknown action %q — valid: stop, start, hibernate, terminate, extend", action)), nil
	}
}
