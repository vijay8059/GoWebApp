// Package auth issues and verifies JWTs.
//
// The current implementation uses HS256 with a shared secret, which is the
// right trade-off for a single-service portfolio repo: zero key-distribution
// overhead and a clear upgrade path. For a multi-service deployment the
// secret should be replaced with an RS256 key pair so that verifiers only
// need the public half — see docs/ARCHITECTURE.md.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingSecret = errors.New("auth: JWT secret is required")
	ErrInvalidToken  = errors.New("auth: invalid token")
)

// Config carries the verifier/signer's expected claims.
type Config struct {
	Secret   string
	Issuer   string
	Audience string
	TTL      time.Duration
}

// Verifier validates and (optionally) signs JWTs.
type Verifier struct {
	cfg    Config
	parser *jwt.Parser
}

// New returns a Verifier, validating the configuration up front so that
// misconfiguration is a startup error rather than a per-request 500.
func New(cfg Config) (*Verifier, error) {
	if cfg.Secret == "" {
		return nil, ErrMissingSecret
	}
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	}
	if cfg.Issuer != "" {
		opts = append(opts, jwt.WithIssuer(cfg.Issuer))
	}
	if cfg.Audience != "" {
		opts = append(opts, jwt.WithAudience(cfg.Audience))
	}
	return &Verifier{cfg: cfg, parser: jwt.NewParser(opts...)}, nil
}

// Claims is the subset of JWT claims we use. Embedding RegisteredClaims gives
// us free validation of exp, nbf, iat, iss, aud.
type Claims struct {
	jwt.RegisteredClaims
}

// Parse verifies the token's signature and claims and returns the subject.
func (v *Verifier) Parse(tokenStr string) (subject string, err error) {
	var claims Claims
	tok, err := v.parser.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
		// SigningMethodHS256 is already enforced via WithValidMethods, but a
		// defence-in-depth check here keeps the keyfunc honest if the parser
		// options ever drift.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %s", ErrInvalidToken, t.Method.Alg())
		}
		return []byte(v.cfg.Secret), nil
	})
	if err != nil || !tok.Valid {
		return "", fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if claims.Subject == "" {
		return "", fmt.Errorf("%w: missing subject", ErrInvalidToken)
	}
	return claims.Subject, nil
}

// Sign mints a token for the given subject. Used by the cmd/token helper and
// by tests; the running server never calls Sign.
func (v *Verifier) Sign(subject string) (string, error) {
	now := time.Now()
	ttl := v.cfg.TTL
	if ttl <= 0 {
		ttl = time.Hour
	}
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    v.cfg.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	if v.cfg.Audience != "" {
		claims.Audience = jwt.ClaimStrings{v.cfg.Audience}
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(v.cfg.Secret))
}
