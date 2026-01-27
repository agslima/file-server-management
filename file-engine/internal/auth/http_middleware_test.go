package auth

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    jwt "github.com/golang-jwt/jwt/v5"
)

func TestHTTPAuthMiddleware(t *testing.T) {
    secret := "test-secret"

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "sub":   "user-1",
        "roles": []string{"viewer"},
        "exp":   time.Now().Add(time.Hour).Unix(),
    })
    signed, _ := token.SignedString([]byte(secret))

    verifier, _ := NewJWTVerifier(secret, "", "", "")

    called := false
    handler := HTTPAuthMiddleware(verifier, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, ok := FromContext(r.Context())
        if !ok {
            t.Fatal("auth context not found")
        }
        if ctx.UserID != "user-1" {
            t.Fatalf("unexpected user id %s", ctx.UserID)
        }
        called = true
        w.WriteHeader(200)
    }))

    req := httptest.NewRequest("GET", "/test", nil)
    req.Header.Set("Authorization", "Bearer "+signed)
    rr := httptest.NewRecorder()

    handler.ServeHTTP(rr, req)

    if !called {
        t.Fatal("handler was not called")
    }
    if rr.Code != 200 {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
}
