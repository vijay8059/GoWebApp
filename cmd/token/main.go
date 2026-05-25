// Command token mints a JWT for local testing.
//
//	JWT_SECRET=dev-secret-32-bytes-or-more-please go run ./cmd/token -sub alice
//
// Prints the token to stdout. Never embed this in a production workflow —
// tokens for real users should come from your IdP / auth service.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/GoWebApp/internal/auth"
)

func main() {
	sub := flag.String("sub", "test-user", "subject claim")
	ttl := flag.Duration("ttl", time.Hour, "token lifetime")
	issuer := flag.String("iss", envOr("JWT_ISSUER", "gowebapp"), "issuer claim")
	audience := flag.String("aud", envOr("JWT_AUDIENCE", "gowebapp"), "audience claim")
	flag.Parse()

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		fmt.Fprintln(os.Stderr, "JWT_SECRET env var is required")
		os.Exit(2)
	}

	v, err := auth.New(auth.Config{
		Secret: secret, Issuer: *issuer, Audience: *audience, TTL: *ttl,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	tok, err := v.Sign(*sub)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(tok)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
