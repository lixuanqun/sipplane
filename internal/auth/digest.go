package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/icholy/digest"
)

// Challenger issues and verifies SIP Digest authentication (RFC 2617 / 3261).
type Challenger struct {
	Realm string

	mu     sync.Mutex
	nonces map[string]time.Time
}

// NewChallenger creates a digest challenger for the given realm.
func NewChallenger(realm string) *Challenger {
	if realm == "" {
		realm = "sipplane"
	}
	return &Challenger{
		Realm:  realm,
		nonces: make(map[string]time.Time),
	}
}

// ChallengeHeader builds a WWW-Authenticate header value.
func (c *Challenger) ChallengeHeader() string {
	nonce := randomNonce()
	c.mu.Lock()
	c.nonces[nonce] = time.Now().Add(5 * time.Minute)
	c.mu.Unlock()
	chal := &digest.Challenge{
		Realm:     c.Realm,
		Nonce:     nonce,
		Algorithm: "MD5",
		QOP:       []string{"auth"},
	}
	return chal.String()
}

// Verify checks Authorization header against username/password/method/uri.
func (c *Challenger) Verify(authHeader, username, password, method, uri string) error {
	if authHeader == "" {
		return fmt.Errorf("missing authorization")
	}
	cred, err := digest.ParseCredentials(authHeader)
	if err != nil {
		return err
	}
	if cred.Username != username {
		return fmt.Errorf("username mismatch")
	}
	c.mu.Lock()
	exp, ok := c.nonces[cred.Nonce]
	c.mu.Unlock()
	if !ok || time.Now().After(exp) {
		return fmt.Errorf("invalid or expired nonce")
	}
	chal := &digest.Challenge{
		Realm:     c.Realm,
		Nonce:     cred.Nonce,
		Algorithm: cred.Algorithm,
		Opaque:    cred.Opaque,
		QOP:       []string{"auth"},
	}
	if cred.QOP == "" {
		chal.QOP = nil
	}
	expected, err := digest.Digest(chal, digest.Options{
		Method:   method,
		URI:      uri,
		Username: username,
		Password: password,
		Cnonce:   cred.Cnonce,
		Count:    cred.Nc,
	})
	if err != nil {
		return err
	}
	if !strings.EqualFold(expected.Response, cred.Response) {
		return fmt.Errorf("bad response digest")
	}
	return nil
}

func randomNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
