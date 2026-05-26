package social

import (
	"strconv"
	"strings"
)

func normalizeRawProfile(raw map[string]any, email string) map[string]any {
	if raw == nil {
		raw = map[string]any{}
	}
	email = strings.TrimSpace(email)
	if email != "" {
		raw["email"] = email
		if domain := emailDomain(email); domain != "" {
			raw["email_domain"] = domain
		}
	}
	return raw
}

func emailDomain(email string) string {
	parts := strings.Split(strings.TrimSpace(strings.ToLower(email)), "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func rawProfileBool(raw map[string]any, key string) (bool, bool) {
	if raw == nil {
		return false, false
	}
	v, ok := raw[key]
	if !ok || v == nil {
		return false, false
	}
	switch value := v.(type) {
	case bool:
		return value, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		return parsed, err == nil
	default:
		return false, false
	}
}
