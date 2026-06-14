package main

import (
	"strings"
	"testing"
)

func TestBuildDiscordEmbed_ColorAndFields(t *testing.T) {
	nr := NotifyRequest{
		Platform:     "discord",
		EventType:    "ttl_warning",
		InstanceName: "rstudio",
		InstanceID:   "i-0abc123",
		Region:       "us-west-2",
		DNSName:      "rstudio.5k0zfnmq.spore.host",
		Detail:       "10m",
	}
	e := buildDiscordEmbed(nr)

	if e.Color != discordColorYellow {
		t.Errorf("ttl_warning color = %#x, want yellow %#x", e.Color, discordColorYellow)
	}
	if !strings.Contains(e.Title, "rstudio") || !strings.Contains(e.Title, "10m") {
		t.Errorf("title = %q, want instance name + detail", e.Title)
	}
	if e.URL != "https://rstudio.5k0zfnmq.spore.host" {
		t.Errorf("url = %q", e.URL)
	}
	// Expect URL, Instance, Region fields.
	var hasInstance, hasRegion bool
	for _, f := range e.Fields {
		if f.Name == "Instance" && strings.Contains(f.Value, "i-0abc123") {
			hasInstance = true
		}
		if f.Name == "Region" && f.Value == "us-west-2" {
			hasRegion = true
		}
	}
	if !hasInstance || !hasRegion {
		t.Errorf("missing Instance/Region fields: %+v", e.Fields)
	}
}

func TestBuildDiscordEmbed_SeverityColors(t *testing.T) {
	cases := map[string]int{
		"completion":     discordColorGreen,
		"ttl_expired":    discordColorRed,
		"idle_stopped":   discordColorRed,
		"spot_interrupt": discordColorYellow,
		"idle_warning":   discordColorYellow,
		"pre_stop_start": discordColorBlue,
	}
	for event, want := range cases {
		e := buildDiscordEmbed(NotifyRequest{EventType: event, InstanceName: "x"})
		if e.Color != want {
			t.Errorf("%s color = %#x, want %#x", event, e.Color, want)
		}
	}
}

func TestBuildDiscordEmbed_UnknownEventDefaultsBlue(t *testing.T) {
	e := buildDiscordEmbed(NotifyRequest{EventType: "something_new", InstanceName: "x"})
	if e.Color != discordColorBlue {
		t.Errorf("unknown event color = %#x, want blue", e.Color)
	}
}
