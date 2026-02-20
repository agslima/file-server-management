package auth

import (
	"testing"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

func TestJWTToAuthContext(t *testing.T) {
	secret := "test-secret"

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, jwtgo.MapClaims{
		"sub":    "user-42",
		"email":  "user42@example.com",
		"groups": []string{"finance", "ops"},
		"roles":  []string{"admin", "editor"},
		"exp":    time.Now().Add(time.Hour).Unix(),
	})

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	verifier, err := NewJWTVerifier(secret, "", "", "")
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	authCtx, err := verifier.ParseAuthContext("Bearer " + signed)
	if err != nil {
		t.Fatalf("parse auth context: %v", err)
	}

	if authCtx.UserID != "user-42" {
		t.Fatalf("expected user-42, got %s", authCtx.UserID)
	}

	if len(authCtx.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(authCtx.Roles))
	}
	if authCtx.EffectiveActorID() != "user-42" || authCtx.Email != "user42@example.com" || len(authCtx.Groups) != 2 {
		t.Fatalf("unexpected normalized identity: %+v", authCtx)
	}
}

func TestJWTActorIDClaimOverride(t *testing.T) {
	secret := "test-secret"
	token := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, jwtgo.MapClaims{
		"sub":      "sub-1",
		"actor_id": "person-77",
		"roles":    []string{"viewer"},
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(secret))

	verifier, err := NewJWTVerifierWithOIDC(secret, "", "", "", "", "actor_id")
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}
	authCtx, err := verifier.ParseAuthContext("Bearer " + signed)
	if err != nil {
		t.Fatalf("parse auth context: %v", err)
	}
	if authCtx.EffectiveActorID() != "person-77" {
		t.Fatalf("expected actor override, got %+v", authCtx)
	}
}

func TestJWTNormalizesKeycloakStyleRoles(t *testing.T) {
	secret := "test-secret"
	token := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, jwtgo.MapClaims{
		"sub": "sub-1",
		"realm_access": map[string]any{
			"roles": []string{"admin", "viewer"},
		},
		"resource_access": map[string]any{
			"file-engine-dev": map[string]any{"roles": []string{"iam-admin"}},
		},
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(secret))

	verifier, err := NewJWTVerifier(secret, "", "", "")
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}
	authCtx, err := verifier.ParseAuthContext("Bearer " + signed)
	if err != nil {
		t.Fatalf("parse auth context: %v", err)
	}
	if len(authCtx.Roles) < 3 {
		t.Fatalf("expected merged roles, got %+v", authCtx.Roles)
	}
}

func TestJWTVerifierWithSecretDoesNotRequireJWKSAtStartup(t *testing.T) {
	verifier, err := NewJWTVerifierWithOIDC(
		"test-secret",
		"",
		"",
		"",
		"http://127.0.0.1:1/realms/file-engine/protocol/openid-connect/certs",
		"sub",
	)
	if err != nil {
		t.Fatalf("new verifier with secret+jwks should not fail startup: %v", err)
	}
	if verifier == nil {
		t.Fatal("expected non-nil verifier")
	}
}

func TestJWTFallsBackWhenSubIsMissing(t *testing.T) {
	secret := "test-secret"
	token := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, jwtgo.MapClaims{
		"preferred_username": "dev-admin",
		"email":              "dev-admin@example.com",
		"roles":              []string{"admin"},
		"exp":                time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	verifier, err := NewJWTVerifier(secret, "", "", "")
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}
	authCtx, err := verifier.ParseAuthContext("Bearer " + signed)
	if err != nil {
		t.Fatalf("parse auth context: %v", err)
	}
	if authCtx.UserID != "dev-admin" {
		t.Fatalf("expected fallback user id dev-admin, got %q", authCtx.UserID)
	}
	if authCtx.EffectiveActorID() != "dev-admin" {
		t.Fatalf("expected actor id fallback to user id, got %+v", authCtx)
	}
}
