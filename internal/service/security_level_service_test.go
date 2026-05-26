package service

import (
	"testing"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/google/uuid"
)

func TestEvaluateConditionLegacyBindingAgeRule(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := newRuleEvalContext(nil, []*domain.SocialBinding{
		{
			ID:       uuid.New(),
			Provider: domain.ProviderGitHub,
			Status:   domain.SocialBindingStatusActive,
			BoundAt:  now.AddDate(0, 0, -45),
		},
	}, now)

	condition := domain.RuleCondition{
		Provider:       domain.ProviderGitHub,
		MinBindingDays: 30,
	}

	if !evaluateCondition(condition, ctx) {
		t.Fatal("expected legacy provider + min_binding_days rule to match active binding age")
	}
}

func TestEvaluateConditionUsesProviderAccountCreatedAt(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := newRuleEvalContext(nil, []*domain.SocialBinding{
		{
			ID:       uuid.New(),
			Provider: domain.ProviderGitHub,
			Status:   domain.SocialBindingStatusActive,
			BoundAt:  now.AddDate(0, 0, -1),
			RawProfile: map[string]any{
				"created_at": "2025-01-01T00:00:00Z",
			},
		},
	}, now)

	condition := domain.RuleCondition{
		Type:     domain.ConditionProviderAccountAgeDays,
		Provider: domain.ProviderGitHub,
		MinDays:  300,
	}

	if !evaluateCondition(condition, ctx) {
		t.Fatal("expected provider_account_age_days to use RawProfile created_at, not binding time")
	}
}

func TestEvaluateConditionProviderAccountCreatedAtMissingFails(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := newRuleEvalContext(nil, []*domain.SocialBinding{
		{
			ID:       uuid.New(),
			Provider: domain.ProviderGitHub,
			Status:   domain.SocialBindingStatusActive,
			BoundAt:  now.AddDate(0, 0, -400),
			RawProfile: map[string]any{
				"followers": 10,
			},
		},
	}, now)

	condition := domain.RuleCondition{
		Type:     domain.ConditionProviderAccountAgeDays,
		Provider: domain.ProviderGitHub,
		MinDays:  300,
	}

	if evaluateCondition(condition, ctx) {
		t.Fatal("expected missing provider created_at to fail instead of falling back to binding time")
	}
}

func TestEvaluateConditionProviderEmailVerifiedUnknownDoesNotMatchFalse(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := newRuleEvalContext(nil, []*domain.SocialBinding{
		{
			ID:       uuid.New(),
			Provider: domain.ProviderGitHub,
			Status:   domain.SocialBindingStatusActive,
			BoundAt:  now.AddDate(0, 0, -30),
			RawProfile: map[string]any{
				"email": "user@example.com",
			},
		},
	}, now)

	condition := domain.RuleCondition{
		Type:     domain.ConditionProviderEmailVerified,
		Provider: domain.ProviderGitHub,
		Operator: "eq",
		Value:    false,
	}

	if evaluateCondition(condition, ctx) {
		t.Fatal("expected unknown email verification state to be non-matching, not false")
	}
}

func TestEvaluateConditionIgnoresInactiveBindings(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := newRuleEvalContext(nil, []*domain.SocialBinding{
		{
			ID:       uuid.New(),
			Provider: domain.ProviderGitHub,
			Status:   domain.SocialBindingStatusUserUnbound,
			BoundAt:  now.AddDate(0, 0, -400),
		},
	}, now)

	condition := domain.RuleCondition{
		Type:     domain.ConditionProviderBound,
		Provider: domain.ProviderGitHub,
	}

	if evaluateCondition(condition, ctx) {
		t.Fatal("expected inactive social binding to be ignored")
	}
}

func TestEvaluateConditionUsesAgeComparisonOperator(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := newRuleEvalContext(&domain.User{CreatedAt: now.AddDate(0, 0, -5)}, []*domain.SocialBinding{
		{
			ID:       uuid.New(),
			Provider: domain.ProviderGitHub,
			Status:   domain.SocialBindingStatusActive,
			BoundAt:  now.AddDate(0, 0, -5),
			RawProfile: map[string]any{
				"created_at": now.AddDate(0, 0, -5).Format(time.RFC3339),
			},
		},
	}, now)

	cases := []struct {
		name      string
		condition domain.RuleCondition
	}{
		{
			name: "binding age lte",
			condition: domain.RuleCondition{
				Type:     domain.ConditionBindingAgeDays,
				Provider: domain.ProviderGitHub,
				Operator: "lte",
				MinDays:  7,
			},
		},
		{
			name: "provider account age lt",
			condition: domain.RuleCondition{
				Type:     domain.ConditionProviderAccountAgeDays,
				Provider: domain.ProviderGitHub,
				Operator: "lt",
				MinDays:  7,
			},
		},
		{
			name: "user age lte",
			condition: domain.RuleCondition{
				Type:     domain.ConditionUserCreatedAgeDays,
				Operator: "lte",
				MinDays:  7,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !evaluateCondition(tc.condition, ctx) {
				t.Fatalf("expected %s to honor comparison operator", tc.name)
			}
		})
	}
}
