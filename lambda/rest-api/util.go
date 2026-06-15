package main

import (
	"encoding/json"
	"os"
	"strings"
)

func parseJSON(body string, v any) error {
	return json.Unmarshal([]byte(body), v)
}

// isProd reports whether the Lambda is running in the production environment,
// per the SPORE_ENV var (set at deploy: "integ" for staging, "production"/"prod"
// for prod). Used to fail closed on security gates that have non-prod escape
// hatches.
func isProd() bool {
	switch strings.ToLower(os.Getenv("SPORE_ENV")) {
	case "production", "prod":
		return true
	default:
		return false
	}
}

func trim(s string) string               { return strings.TrimSpace(s) }
func splitString(s, sep string) []string { return strings.Split(s, sep) }
