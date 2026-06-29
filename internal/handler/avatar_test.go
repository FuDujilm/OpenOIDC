package handler

import (
	"testing"

	"github.com/anthropic/oidc-platform/internal/domain"
)

func TestResolvedAvatarURLUsesExplicitAvatar(t *testing.T) {
	u := &domain.User{
		Email:     "user@example.com",
		AvatarURL: " https://example.com/avatar.png ",
	}

	if got, want := resolvedAvatarURL(u), "https://example.com/avatar.png"; got != want {
		t.Fatalf("resolvedAvatarURL() = %q, want %q", got, want)
	}
}

func TestResolvedAvatarURLFallsBackToGravatar(t *testing.T) {
	u := &domain.User{Email: " User@Example.COM "}

	if got, want := resolvedAvatarURL(u), "https://gravatar.loli.net/avatar/b58996c504c5638798eb6b511e6f49af"; got != want {
		t.Fatalf("resolvedAvatarURL() = %q, want %q", got, want)
	}
}
