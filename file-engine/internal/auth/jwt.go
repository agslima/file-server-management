package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"
	"strings"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

// Claims shape.
// Recommended:
// - sub: user id
// - roles: ["admin","viewer"]
// - iss, aud as needed
type Claims struct {
	Roles []string `json:"roles"`
	jwtgo.RegisteredClaims
}

// JWTVerifier validates JWTs using either:
// - HMAC secret (JWTSecret) or
// - RSA public key PEM (JWTPublicKeyPEM)
type JWTVerifier struct {
	secret   []byte
	pubKey   *rsa.PublicKey
	issuer   string
	audience string
}

func NewJWTVerifier(secret, publicKeyPEM, issuer, audience string) (*JWTVerifier, error) {
	v := &JWTVerifier{issuer: issuer, audience: audience}
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
	if len(v.secret) == 0 && v.pubKey == nil {
		return nil, errors.New("JWT verifier requires JWT_SECRET or JWT_PUBLIC_KEY_PEM")
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
		// enforce signing method
		switch t.Method.(type) {
		case *jwtgo.SigningMethodHMAC:
			if len(v.secret) == 0 {
				return nil, errors.New("hmac not configured")
			}
			return v.secret, nil
		case *jwtgo.SigningMethodRSA:
			if v.pubKey == nil {
				return nil, errors.New("rsa public key not configured")
			}
			return v.pubKey, nil
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

	// Optional issuer/audience checks
	if v.issuer != "" && claims.Issuer != v.issuer {
		return AuthContext{}, errors.New("invalid issuer")
	}
	if v.audience != "" {
		audOK := slices.Contains(claims.Audience, v.audience)
		if !audOK {
			return AuthContext{}, errors.New("invalid audience")
		}
	}

	userID := ""
	if claims.Subject != "" {
		userID = claims.Subject
	}
	if userID == "" {
		return AuthContext{}, errors.New("missing sub claim")
	}

	return AuthContext{UserID: userID, Roles: claims.Roles}, nil
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
	// Try parsing certificate
	cert, err2 := x509.ParseCertificate(block.Bytes)
	if err2 == nil {
		if pk, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			return pk, nil
		}
		return nil, errors.New("cert public key is not RSA")
	}
	return nil, err
}
