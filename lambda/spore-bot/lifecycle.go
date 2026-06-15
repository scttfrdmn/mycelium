package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// maxBotTTL is the hard ceiling on the resulting TTL/idle timeout through the
// bot (spore-host#371) — keeps a runaway `/spore extend foo 99999h` from
// disabling the lifecycle backstop.
const maxBotTTL = 7 * 24 * time.Hour

// computeExtendedDeadline returns the new absolute TTL deadline for an extend.
// It pushes the existing deadline forward by extension; if there is no deadline
// tag it anchors to launchTime+newTTL. A safety floor (2026-06 audit, M-corr)
// guarantees the result is never earlier than now+extension — so a missing or
// already-expired prior deadline (or a stale launch anchor) can never set a
// deadline in the past and reap the instance the moment the user asks to keep
// it. An extend always grants at least `extension` from the current moment.
func computeExtendedDeadline(now, currentDeadline, launchTime time.Time, newTTL, extension time.Duration) time.Time {
	var d time.Time
	if currentDeadline.IsZero() {
		d = launchTime.Add(newTTL)
	} else {
		d = currentDeadline.Add(extension)
	}
	if floor := now.Add(extension); d.Before(floor) {
		d = floor
	}
	return d
}

// extendTTL adds duration to the instance's current TTL by updating the spawn:ttl EC2 tag.
// Usage: /spore extend <name> <duration>  e.g. /spore extend rstudio 2h
func extendTTL(ctx context.Context, client *ec2.Client, reg *BotRegistration, durationStr, slashCmd string) (string, error) {
	if durationStr == "" {
		return fmt.Sprintf("Usage: `%s extend <name> <duration>`\nExample: `%s extend %s 2h`",
			slashCmd, slashCmd, reg.Nickname), nil
	}

	extension, err := time.ParseDuration(durationStr)
	if err != nil || extension <= 0 {
		return fmt.Sprintf("❌ Invalid duration `%s`. Use a format like `2h`, `30m`, or `1h30m`.", durationStr), nil
	}

	// Get current TTL + deadline tags
	out, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{reg.InstanceID},
	})
	if err != nil || len(out.Reservations) == 0 || len(out.Reservations[0].Instances) == 0 {
		return "", fmt.Errorf("describe instance: %w", err)
	}
	inst := out.Reservations[0].Instances[0]

	var currentTTL time.Duration
	var launchTime time.Time
	var currentDeadline time.Time
	if inst.LaunchTime != nil {
		launchTime = *inst.LaunchTime
	}
	for _, tag := range inst.Tags {
		if tag.Key == nil || tag.Value == nil {
			continue
		}
		switch *tag.Key {
		case reg.TagPrefix + ":ttl":
			currentTTL, _ = time.ParseDuration(*tag.Value)
		case reg.TagPrefix + ":ttl-deadline":
			currentDeadline, _ = time.Parse(time.RFC3339, *tag.Value)
		}
	}

	// New TTL = existing TTL + extension (relative to launch time)
	newTTL := currentTTL + extension
	if newTTL > maxBotTTL {
		return fmt.Sprintf("❌ Resulting TTL %s exceeds the maximum of %s. Pick a shorter extension.",
			formatTTLDuration(newTTL), formatTTLDuration(maxBotTTL)), nil
	}
	newTTLStr := formatTTLDuration(newTTL)

	// spored treats <prefix>:ttl-deadline (absolute) as authoritative and ignores
	// <prefix>:ttl. Writing only :ttl is a silent no-op — the instance still dies
	// at its original deadline (same bug class as spore-host-mcp#11). Push the
	// absolute deadline forward and write BOTH tags.
	newDeadline := computeExtendedDeadline(time.Now(), currentDeadline, launchTime, newTTL, extension)

	_, err = client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{reg.InstanceID},
		Tags: []ec2types.Tag{
			{Key: aws.String(reg.TagPrefix + ":ttl"), Value: aws.String(newTTLStr)},
			{Key: aws.String(reg.TagPrefix + ":ttl-deadline"), Value: aws.String(newDeadline.UTC().Format(time.RFC3339))},
		},
	})
	if err != nil {
		return "", fmt.Errorf("update TTL tag: %w", err)
	}

	remaining := time.Until(newDeadline)

	return fmt.Sprintf("⏱️ Extended *%s* TTL by %s.\n  New deadline: %s (%s remaining)",
		reg.Nickname,
		durationStr,
		newDeadline.UTC().Format("2 Jan 15:04 UTC"),
		formatHMS(remaining)), nil
}

// setIdleTimeout updates (or removes) the spawn:idle-timeout EC2 tag.
// Usage: /spore idle <name> <duration|off>  e.g. /spore idle rstudio 30m
func setIdleTimeout(ctx context.Context, client *ec2.Client, reg *BotRegistration, durationStr, slashCmd string) (string, error) {
	if durationStr == "" {
		return fmt.Sprintf("Usage: `%s idle <name> <duration|off>`\nExamples: `%s idle %s 30m` or `%s idle %s off`",
			slashCmd, slashCmd, reg.Nickname, slashCmd, reg.Nickname), nil
	}

	if durationStr == "off" || durationStr == "none" || durationStr == "disable" {
		// Remove the idle timeout tag
		_, err := client.DeleteTags(ctx, &ec2.DeleteTagsInput{
			Resources: []string{reg.InstanceID},
			Tags: []ec2types.Tag{
				{Key: aws.String(reg.TagPrefix + ":idle-timeout")},
			},
		})
		if err != nil {
			return "", fmt.Errorf("remove idle timeout tag: %w", err)
		}
		return fmt.Sprintf("💤 Idle timeout disabled for *%s*.", reg.Nickname), nil
	}

	d, err := time.ParseDuration(durationStr)
	if err != nil || d <= 0 {
		return fmt.Sprintf("❌ Invalid duration `%s`. Use a format like `30m`, `1h`, or `off` to disable.", durationStr), nil
	}
	if d > maxBotTTL {
		return fmt.Sprintf("❌ Idle timeout %s exceeds the maximum of %s.", durationStr, formatTTLDuration(maxBotTTL)), nil
	}

	_, err = client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{reg.InstanceID},
		Tags: []ec2types.Tag{
			{Key: aws.String(reg.TagPrefix + ":idle-timeout"), Value: aws.String(durationStr)},
		},
	})
	if err != nil {
		return "", fmt.Errorf("update idle timeout tag: %w", err)
	}

	return fmt.Sprintf("💤 *%s* will stop after %s of inactivity.", reg.Nickname, durationStr), nil
}

// formatTTLDuration formats a duration as a clean string for the spawn:ttl tag.
func formatTTLDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d.Hours() >= 1 && d == d.Round(time.Hour) {
		return fmt.Sprintf("%.0fh", d.Hours())
	}
	if d.Minutes() >= 1 && d == d.Round(time.Minute) {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}
