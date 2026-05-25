package social

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/oauth2"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

func NewCustomOAuth2Provider(name, clientID, clientSecret string, custom domain.CustomOAuth2Config) *OAuth2Provider {
	if custom.AuthURL == "" || custom.TokenURL == "" || custom.UserURL == "" {
		return nil
	}
	return &OAuth2Provider{
		name: name,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  custom.AuthURL,
				TokenURL: custom.TokenURL,
			},
			Scopes: custom.Scopes,
		},
		userURL: custom.UserURL,
		parseUser: func(body []byte) (*port.ProviderUserInfo, error) {
			var raw map[string]any
			if err := json.Unmarshal(body, &raw); err != nil {
				return nil, fmt.Errorf("decode custom oauth2 user: %w", err)
			}
			uid := valueAtPath(raw, custom.IDPath)
			if uid == "" {
				return nil, fmt.Errorf("custom oauth2 user missing id at %q", custom.IDPath)
			}
			return &port.ProviderUserInfo{
				ProviderUID:   uid,
				Email:         valueAtPath(raw, custom.EmailPath),
				EmailVerified: boolAtPath(raw, "email_verified"),
				DisplayName:   valueAtPath(raw, custom.NamePath),
				AvatarURL:     valueAtPath(raw, custom.AvatarPath),
				RawProfile:    raw,
			}, nil
		},
	}
}

func valueAtPath(data map[string]any, path string) string {
	if path == "" {
		return ""
	}
	var current any = data
	for _, part := range strings.Split(path, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[part]
	}
	switch v := current.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func boolAtPath(data map[string]any, path string) bool {
	v := strings.ToLower(valueAtPath(data, path))
	return v == "true" || v == "1" || v == "yes"
}
