package main

import "testing"

// TestInstanceFQDN covers the #121 display-FQDN selection: prefer the friendly
// account-name segment, fall back to base36, and degrade to "" when there's
// nothing to build from.
func TestInstanceFQDN(t *testing.T) {
	cases := []struct {
		name, short, acctName, base36, want string
	}{
		{"prefers account-name", "job", "mycelium-development", "5k0zfnmq", "job.mycelium-development.spore.host"},
		{"falls back to base36", "job", "", "5k0zfnmq", "job.5k0zfnmq.spore.host"},
		{"no short name -> empty", "", "mycelium-development", "5k0zfnmq", ""},
		{"no segment -> empty", "job", "", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := instanceFQDN(c.short, c.acctName, c.base36); got != c.want {
				t.Errorf("instanceFQDN(%q,%q,%q) = %q, want %q", c.short, c.acctName, c.base36, got, c.want)
			}
		})
	}
}
