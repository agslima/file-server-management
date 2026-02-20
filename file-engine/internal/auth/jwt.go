package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Roles             []string `json:"roles"`
	Email             string   `json:"email"`
	ActorID           string   `json:"actor_id"`
	PreferredUsername string   `json:"preferred_username"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`
	// OIDC group claims typically uses `groups`.
	Groups []string `json:"groups"`
	jwtgo.RegisteredClaims
}

type jwksResolver struct {
	url    string
	client *http.Client
	keys   map[string]*rsa.PublicKey
}

type JWTVerifier struct {
	secret       []byte
	pubKey       *rsa.PublicKey
	jwks         *jwksResolver
	issuer       string
	audience     string
	actorIDClaim string
}

func NewJWTVerifier(secret, publicKeyPEM, issuer, audience string) (*JWTVerifier, error) {
	return NewJWTVerifierWithOIDC(secret, publicKeyPEM, issuer, audience, "", "sub")
}

func NewJWTVerifierWithOIDC(secret, publicKeyPEM, issuer, audience, jwksURL, actorIDClaim string) (*JWTVerifier, error) {
	v := &JWTVerifier{issuer: issuer, audience: audience, actorIDClaim: strings.TrimSpace(actorIDClaim)}
	if v.actorIDClaim == "" {
		v.actorIDClaim = "sub"
	}
	if secret != "" {
		v.secret = []byte(secret)
	}
	if publicKeyPEM != "" {
		pk, err := parseRSAPublicKeyFromPEM(publicKeyPEM)
		if err != nil {
			return nil, err
		}
		v.pubKey = pk
	}
	if strings.TrimSpace(jwksURL) != "" {
		resolver := &jwksResolver{
			url:    strings.TrimSpace(jwksURL),
			client: &http.Client{Timeout: 2 * time.Second},
			keys:   map[string]*rsa.PublicKey{},
		}
		if err := resolver.refresh(); err != nil {
			return nil, fmt.Errorf("load jwks: %w", err)
		}
		v.jwks = resolver
	}
	if len(v.secret) == 0 && v.pubKey == nil && v.jwks == nil {
		return nil, errors.New("JWT verifier requires JWT_SECRET or JWT_PUBLIC_KEY_PEM or JWT_JWKS_URL")
	}
	return v, nil
}

func (v *JWTVerifier) ParseAuthContext(authHeader string) (AuthContext, error) {
	token := strings.TrimSpace(authHeader)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	if token == "" {
		return AuthContext{}, errors.New("missing token")
	}

	claims := &Claims{}
	parsed, err := jwtgo.ParseWithClaims(token, claims, func(t *jwtgo.Token) (any, error) {
		switch t.Method.(type) {
		case *jwtgo.SigningMethodHMAC:
			if len(v.secret) == 0 {
				return nil, errors.New("hmac not configured")
			}
			return v.secret, nil
		case *jwtgo.SigningMethodRSA:
			if v.pubKey != nil {
				return v.pubKey, nil
			}
			if v.jwks != nil {
				kid, _ := t.Header["kid"].(string)
				return v.jwks.keyForKID(kid)
			}
			return nil, errors.New("rsa public key not configured")
		default:
			return nil, fmt.Errorf("unsupported signing method: %s", t.Method.Alg())
		}
	})
	if err != nil {
		return AuthContext{}, err
	}
	if !parsed.Valid {
		return AuthContext{}, errors.New("invalid token")
	}
	if v.issuer != "" && claims.Issuer != v.issuer {
		return AuthContext{}, errors.New("invalid issuer")
	}
	if v.audience != "" {
		if !slices.Contains(claims.Audience, v.audience) {
			return AuthContext{}, errors.New("invalid audience")
		}
	}
	if claims.Subject == "" {
		return AuthContext{}, errors.New("missing sub claim")
	}
	actorID := strings.TrimSpace(resolveActorID(v.actorIDClaim, claims))
	if actorID == "" {
		actorID = claims.Subject
	}
	roles := normalizedRoles(claims)
	return AuthContext{UserID: claims.Subject, ActorID: actorID, Email: claims.Email, Groups: claims.Groups, Roles: roles}, nil
}

func resolveActorID(actorIDClaim string, claims *Claims) string {
	switch strings.TrimSpace(strings.ToLower(actorIDClaim)) {
	case "sub", "subject", "":
		return claims.Subject
	case "email":
		return claims.Email
	case "actor_id":
		return claims.ActorID
	case "preferred_username", "username":
		return claims.PreferredUsername
	default:
		return claims.Subject
	}
}

func normalizedRoles(claims *Claims) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	add := func(roles []string) {
		for _, role := range roles {
			role = strings.TrimSpace(role)
			if role == "" {
				continue
			}
			if _, ok := seen[role]; ok {
				continue
			}
			seen[role] = struct{}{}
			out = append(out, role)
		}
	}
	add(claims.Roles)
	add(claims.RealmAccess.Roles)
	for _, v := range claims.ResourceAccess {
		add(v.Roles)
	}
	return out
}

type jwksDocument struct {
	Keys []jsonWebKey `json:"keys"`
}

type jsonWebKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (r *jwksResolver) keyForKID(kid string) (*rsa.PublicKey, error) {
	if key, ok := r.keys[kid]; ok {
		return key, nil
	}
	if err := r.refresh(); err != nil {
		return nil, err
	}
	if key, ok := r.keys[kid]; ok {
		return key, nil
	}
	return nil, fmt.Errorf("kid %q not found in jwks", kid)
}

func (r *jwksResolver) refresh() error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, r.url, http.NoBody) // #nosec G107 -- URL comes from controlled env config.
	if err != nil {
		return err
	}
	resp, err := r.client.Do(req) // #nosec G704 -- request URL is controlled by trusted env configuration.
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks status %d", resp.StatusCode)
	}
	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return err
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kty != "RSA" || k.Kid == "" {
			continue
		}
		pub, err := parseRSAFromJWK(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("jwks did not contain RSA keys")
	}
	r.keys = keys
	return nil
}

func parseRSAFromJWK(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

func parseRSAPublicKeyFromPEM(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to decode PEM")
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		if pk, ok := pubAny.(*rsa.PublicKey); ok {
			return pk, nil
		}
		return nil, errors.New("PEM is not RSA public key")
	}
	cert, err2 := x509.ParseCertificate(block.Bytes)
	if err2 == nil {
		if pk, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			return pk, nil
		}
		return nil, errors.New("cert public key is not RSA")
	}
	return nil, err
}
