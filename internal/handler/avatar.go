package handler

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"github.com/anthropic/oidc-platform/internal/domain"
)

const gravatarBaseURL = "https://gravatar.loli.net/avatar/"

func resolvedAvatarURL(u *domain.User) string {
	if u == nil {
		return ""
	}
	if avatar := strings.TrimSpace(u.AvatarURL); avatar != "" {
		return avatar
	}
	email := strings.ToLower(strings.TrimSpace(u.Email))
	if email == "" {
		return ""
	}
	sum := md5.Sum([]byte(email))
	return gravatarBaseURL + hex.EncodeToString(sum[:])
}
