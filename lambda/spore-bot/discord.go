package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Discord embed colors (decimal RGB), keyed by event severity (#2).
const (
	discordColorGreen  = 0x2ECC71 // completion / healthy
	discordColorYellow = 0xF1C40F // warnings (ttl/idle warn, spot interrupt)
	discordColorRed    = 0xE74C3C // terminated / stopped
	discordColorBlue   = 0x3498DB // informational / in-progress
)

// discordEmbed is the subset of Discord's embed object we populate.
type discordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	URL         string              `json:"url,omitempty"`
	Fields      []discordEmbedField `json:"fields,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// buildDiscordEmbed turns a lifecycle notification into a color-coded Discord
// embed: title summarizes the event, fields carry instance/region/URL (#2).
func buildDiscordEmbed(nr NotifyRequest) discordEmbed {
	name := nr.InstanceName
	if name == "" {
		name = nr.InstanceID
	}

	var icon, title string
	color := discordColorBlue
	switch nr.EventType {
	case "ttl_warning":
		icon, title, color = "⏱️", fmt.Sprintf("%s terminates in %s", name, nr.Detail), discordColorYellow
	case "ttl_expired":
		icon, title, color = "🔴", fmt.Sprintf("%s has terminated — scheduled end time reached", name), discordColorRed
	case "idle_warning":
		icon, title, color = "💤", fmt.Sprintf("%s will stop in %s — no activity detected", name, nr.Detail), discordColorYellow
	case "idle_stopped":
		icon, title, color = "⏹️", fmt.Sprintf("%s has stopped — idle timeout reached", name), discordColorRed
	case "idle_hibernated":
		icon, title, color = "💤", fmt.Sprintf("%s has hibernated — idle timeout reached", name), discordColorRed
	case "idle_terminated":
		icon, title, color = "🔴", fmt.Sprintf("%s has terminated — idle timeout reached", name), discordColorRed
	case "completion":
		icon, title, color = "✅", fmt.Sprintf("%s has completed", name), discordColorGreen
	case "spot_interrupt":
		icon, title, color = "⚠️", fmt.Sprintf("%s received a Spot interruption notice — %s", name, nr.Detail), discordColorYellow
	case "pre_stop_start":
		icon, title, color = "🔄", fmt.Sprintf("%s is running its shutdown task before terminating", name), discordColorBlue
	default:
		icon, title = "ℹ️", fmt.Sprintf("%s: %s", name, nr.EventType)
	}

	embed := discordEmbed{
		Title:     icon + " " + title,
		Color:     color,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if nr.DNSName != "" {
		embed.URL = "https://" + nr.DNSName
		embed.Fields = append(embed.Fields, discordEmbedField{Name: "URL", Value: "https://" + nr.DNSName, Inline: false})
	}
	if nr.InstanceID != "" {
		embed.Fields = append(embed.Fields, discordEmbedField{Name: "Instance", Value: "`" + nr.InstanceID + "`", Inline: true})
	}
	if nr.Region != "" {
		embed.Fields = append(embed.Fields, discordEmbedField{Name: "Region", Value: nr.Region, Inline: true})
	}
	return embed
}

// postDiscordWebhook POSTs an embed to a Discord channel webhook URL (#2).
func postDiscordWebhook(webhookURL string, embed discordEmbed) {
	payload, _ := json.Marshal(map[string]interface{}{
		"embeds": []discordEmbed{embed},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(payload))
	if err != nil {
		logf("discord webhook request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		logf("discord webhook call error: %v", err)
		return
	}
	defer resp.Body.Close()
	// Discord returns 204 No Content on success for webhook posts.
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		logf("discord webhook returned %d", resp.StatusCode)
	}
}
