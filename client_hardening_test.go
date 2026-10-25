package amaquet

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/newfoundcodes/amaquet-go-sdk/internal/protocol"
)

func TestTimedOutRequestDoesNotLeakPendingEntry(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	c := &Client{
		conn:    clientConn,
		Timeout: 25 * time.Millisecond,
		pending: map[uint64]chan protocol.Frame{},
		subs:    map[uint64]chan PubSubMessage{},
		closed:  make(chan struct{}),
	}
	defer c.Close()

	go func() {
		r := bufio.NewReader(serverConn)
		_, _ = protocol.ReadBufferedFrame(r)

		time.Sleep(100 * time.Millisecond)
	}()

	err := c.request(context.Background(), protocol.OpCommand, map[string]any{
		"command": "PING",
	}, nil, nil)

	if err == nil {
		t.Fatal("expected timeout")
	}
	c.pendingMu.Lock()

	n := len(c.pending)
	c.pendingMu.Unlock()

	if n != 0 {
		t.Fatalf("pending leak: %d", n)
	}
}

func TestDialRejectsURIEmbeddedSecretByDefault(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := Dial(ctx, "amaquet://127.0.0.1:1?api_key=secret")
	if err == nil || !strings.Contains(err.Error(), "API keys in amaquet:// URIs are disabled") {
		t.Fatalf("expected URI-secret rejection, got %v", err)
	}
}

func TestIdentityFingerprintPinningParity(t *testing.T) {
	fake := newFakeNative(t, false)
	ctx := context.Background()

	c, err := DialWithOptions(ctx, fake.url(), DialOptions{
		Timeout:         2 * time.Second,
		RequireIdentity: true,
	})

	if err != nil {
		t.Fatal(err)
	}

	info := c.ServerInfo()
	_ = c.Close()

	computed, ok := info.computedFingerprint()
	if !ok {
		t.Fatal("expected computed Ed25519 fingerprint")
	}

	verified, err := DialWithOptions(ctx, fake.url(), DialOptions{
		Timeout:                     2 * time.Second,
		ExpectedIdentityFingerprint: computed,
	})

	if err != nil {
		t.Fatalf("valid fingerprint pin rejected: %v", err)
	}

	_ = verified.Close()
	_, err = DialWithOptions(ctx, fake.url(), DialOptions{
		Timeout:                     2 * time.Second,
		ExpectedIdentityFingerprint: "ed25519:" + strings.Repeat("0", 64),
	})

	if err == nil {
		t.Fatal("incorrect fingerprint pin accepted")
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	challenge := "test identity nonce"
	self := ServerInfo{
		IdentityPublicKey: base64.RawStdEncoding.EncodeToString(pub),
		NonceSignature:    base64.RawStdEncoding.EncodeToString(ed25519.Sign(priv, []byte(challenge))),
	}

	if !self.VerifyNonce(challenge) {
		t.Fatal("valid Ed25519 signature rejected")
	}

	self.IdentityFingerprint = "ed25519:" + strings.Repeat("0", 64)
	if self.VerifyNonce(challenge) {
		t.Fatal("identity fingerprint was accepted despite a valid signature and mismatching public key")
	}
}

// The generic GET/Command APIs must not silently round 64-bit values when
// callers decode into map[string]any instead of KeySnapshot.
func TestGenericResponsePreservesExactIntegers(t *testing.T) {
	c := dialFake(t, newFakeNative(t, false))
	out, err := c.Get(context.Background(), "big")

	if err != nil {
		t.Fatal(err)
	}

	number, ok := out["value"].(json.Number)
	if !ok || number.String() != "18446744073709551615" {
		t.Fatalf("Get lost exact integer precision: %#v", out["value"])
	}

	var generic map[string]any
	if err := c.Command(context.Background(), "GET", map[string]any{
		"key": "big",
	}, &generic); err != nil {
		t.Fatal(err)
	}

	other, ok := generic["value"].(json.Number)
	if !ok || other.String() != number.String() {
		t.Fatalf("generic Command lost precision: %#v", generic["value"])
	}
}
