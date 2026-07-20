package auth

import (
	"testing"

	"github.com/icholy/digest"
)

func TestDigestChallengeAndVerify(t *testing.T) {
	c := NewChallenger("sipplane")
	hdr := c.ChallengeHeader()
	chal, err := digest.ParseChallenge(hdr)
	if err != nil {
		t.Fatal(err)
	}
	cred, err := digest.Digest(chal, digest.Options{
		Method:   "REGISTER",
		URI:      "sip:sipplane.local",
		Username: "alice",
		Password: "secret",
		Count:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Verify(cred.String(), "alice", "secret", "REGISTER", "sip:sipplane.local"); err != nil {
		t.Fatal(err)
	}
	if err := c.Verify(cred.String(), "alice", "wrong", "REGISTER", "sip:sipplane.local"); err == nil {
		t.Fatal("expected failure for wrong password")
	}
}
