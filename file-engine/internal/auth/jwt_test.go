package auth

import (
    "testing"
    "time"

    jwt "github.com/golang-jwt/jwt/v5"
)

func TestJWTToAuthContext(t *testing.T) {
    secret := "test-secret"

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub":   "user-42",
        "roles": []string{"admin", "editor"},
        "exp":   time.Now().Add(time.Hour).Unix(),
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
}
