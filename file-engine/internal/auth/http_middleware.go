package auth

import (
    "net/http"
)

// HTTPAuthMiddleware extracts Authorization: Bearer <jwt> and stores AuthContext in request context.
func HTTPAuthMiddleware(verifier *JWTVerifier, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authz := r.Header.Get("Authorization")
        a, err := verifier.ParseAuthContext(authz)
        if err != nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        r = r.WithContext(WithAuthContext(r.Context(), a))
        next.ServeHTTP(w, r)
    })
}
