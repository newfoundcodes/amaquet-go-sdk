package amaquet

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/newfoundcodes/amaquet-go-sdk/internal/protocol"
)

// Client: Provides the Client operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Manages native Amaquet TCP/TLS connections, request identifiers, authentication, backpressure, pending replies, subscriptions, and cleanup.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type Client struct {
	endpoint   protocol.Endpoint
	serverInfo ServerInfo
	conn       net.Conn
	writeMu    sync.Mutex
	next       atomic.Uint64
	pendingMu  sync.Mutex
	subsMu     sync.Mutex
	closeOnce  sync.Once

	// Timeout is the default timeout applied to requests made by the client.
	Timeout time.Duration
	// TLSConfig is the TLS configuration used when dialing a secure endpoint.
	TLSConfig *tls.Config

	pending    map[uint64]chan protocol.Frame
	subs       map[uint64]chan PubSubMessage
	subTargets map[string]uint64
	closed     chan struct{}
}

// ServerInfo: Provides the Server Info operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Manages native Amaquet TCP/TLS connections, request identifiers, authentication, backpressure, pending replies, subscriptions, and cleanup.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type ServerInfo struct {
	// Name is the server name reported during the handshake.
	Name string `json:"name"`
	// Version is the server version reported during the handshake.
	Version string `json:"version"`
	// Protocol is the server's native protocol version.
	Protocol int `json:"protocol"`
	// SelectedProtocol is the protocol version selected for this connection.
	SelectedProtocol int `json:"selected_protocol,omitempty"`
	// Capabilities lists optional server capabilities advertised during the handshake.
	Capabilities []string `json:"capabilities,omitempty"`
	// AuthRequired reports whether the server requires authentication.
	AuthRequired bool `json:"auth_required"`
	// IdentityPublicKey is the server identity key encoded with base64.RawStdEncoding.
	IdentityPublicKey string `json:"identity_public_key,omitempty"`
	// IdentityFingerprint is the server identity fingerprint.
	IdentityFingerprint string `json:"identity_fingerprint,omitempty"`
	// NonceSignature is the signed handshake nonce returned by the server.
	NonceSignature string `json:"nonce_signature,omitempty"`
}

// PubSubMessage: Provides the Pub Sub Message operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Manages native Amaquet TCP/TLS connections, request identifiers, authentication, backpressure, pending replies, subscriptions, and cleanup.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type PubSubMessage struct {
	// Channel is the pub/sub channel that carried the message.
	Channel string `json:"channel"`
	// Payload is the decoded message payload.
	Payload any `json:"payload"`
	// PublishedAt is the server timestamp associated with the message.
	PublishedAt time.Time `json:"published_at"`
}

// Subscription: Provides the Subscription operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Manages native Amaquet TCP/TLS connections, request identifiers, authentication, backpressure, pending replies, subscriptions, and cleanup.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type Subscription struct {
	// C receives messages delivered to the subscription.
	C     <-chan PubSubMessage
	close func() error
}

// DialOptions: Provides the Dial Options operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Manages native Amaquet TCP/TLS connections, request identifiers, authentication, backpressure, pending replies, subscriptions, and cleanup.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type DialOptions struct {
	// APIKey is the credential sent during the native handshake.
	APIKey string
	// TLSConfig configures TLS for amaquets:// endpoints.
	TLSConfig *tls.Config
	// Timeout is the default request timeout for the connection.
	Timeout time.Duration
	// AllowURISecrets permits API keys embedded in the connection URI.
	AllowURISecrets bool
	// RequireIdentity rejects servers that do not sign the dial challenge.
	RequireIdentity bool
	// ExpectedIdentityKey pins the base64.RawStdEncoding Ed25519 public key.
	ExpectedIdentityKey string
	// ExpectedIdentityFingerprint pins ed25519:<sha256-hex-of-public-key>.
	ExpectedIdentityFingerprint string
}

// Close: Release transport resources; outstanding requests or subscriptions may stop receiving replies after closure.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (s *Subscription) Close() error {
	if s == nil || s.close == nil {
		return nil
	}

	return s.close()
}

// Dial: Open an Amaquet connection using the selected endpoint, TLS settings, authentication key, and deadline.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `uri`: Native Amaquet URI. Use amaquets:// for TLS.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func Dial(ctx context.Context, uri string) (*Client, error) {
	return DialWithOptions(ctx, uri, DialOptions{})
}

// DialWithOptions: Open an Amaquet connection with explicit TLS, identity, authentication, and timeout options.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `uri`: Native Amaquet URI. Use amaquets:// for TLS.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func DialWithOptions(ctx context.Context, uri string, opts DialOptions) (*Client, error) {
	dialLimit := opts.Timeout
	if dialLimit <= 0 {
		dialLimit = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, dialLimit)
	defer cancel()

	ep, err := protocol.ParseEndpoint(uri)
	if err != nil {
		return nil, err
	}

	if ep.APIKey != "" && !opts.AllowURISecrets && opts.APIKey == "" {
		return nil, fmt.Errorf("API keys in amaquet:// URIs are disabled; pass DialOptions.APIKey")
	}

	apiKey := opts.APIKey
	if apiKey == "" {
		apiKey = ep.APIKey
	}

	d := net.Dialer{Timeout: opts.Timeout}
	if d.Timeout <= 0 {
		d.Timeout = 30 * time.Second
	}

	conn, err := d.DialContext(ctx, "tcp", ep.Address())
	if err != nil {
		return nil, err
	}

	if ep.TLS {
		cfg := opts.TLSConfig
		if cfg == nil {
			cfg = &tls.Config{ServerName: ep.Host, MinVersion: tls.VersionTLS12}
		} else {
			cfg = cfg.Clone()
			if cfg.ServerName == "" {
				cfg.ServerName = ep.Host
			}

			if cfg.MinVersion < tls.VersionTLS12 {
				cfg.MinVersion = tls.VersionTLS12
			}
		}

		tc := tls.Client(conn, cfg)
		if err = tc.HandshakeContext(ctx); err != nil {
			conn.Close()
			return nil, err
		}

		conn = tc
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	c := &Client{
		endpoint:   ep,
		conn:       conn,
		Timeout:    timeout,
		TLSConfig:  opts.TLSConfig,
		pending:    map[uint64]chan protocol.Frame{},
		subs:       map[uint64]chan PubSubMessage{},
		subTargets: map[string]uint64{},
		closed:     make(chan struct{}),
	}
	go c.readLoop()

	var challenge [32]byte
	if _, err = rand.Read(challenge[:]); err != nil {
		c.Close()
		return nil, err
	}

	nonce := base64.RawStdEncoding.EncodeToString(challenge[:])

	var info ServerInfo
	if err = c.request(ctx, protocol.OpHello, map[string]any{
		"nonce":        nonce,
		"min_protocol": int(protocol.Version),
		"max_protocol": int(protocol.Version),
	}, &info, nil); err != nil {
		c.Close()
		return nil, err
	}

	if info.SelectedProtocol != 0 && info.SelectedProtocol != int(protocol.Version) {
		c.Close()
		return nil, fmt.Errorf("server selected unsupported protocol %d", info.SelectedProtocol)
	}

	if info.IdentityPublicKey != "" || info.NonceSignature != "" {
		if !info.VerifyNonce(nonce) {
			c.Close()
			return nil, fmt.Errorf("invalid Amaquet identity signature")
		}
	} else if opts.RequireIdentity ||
		opts.ExpectedIdentityKey != "" ||
		opts.ExpectedIdentityFingerprint != "" {
		c.Close()
		return nil, fmt.Errorf("server identity signature required")
	}

	if opts.ExpectedIdentityKey != "" && subtle.ConstantTimeCompare(
		[]byte(opts.ExpectedIdentityKey),
		[]byte(info.IdentityPublicKey),
	) != 1 {
		c.Close()
		return nil, fmt.Errorf("server identity public key does not match pinned key")
	}

	if opts.ExpectedIdentityFingerprint != "" {
		fingerprint, ok := info.computedFingerprint()
		if !ok || subtle.ConstantTimeCompare([]byte(fingerprint), []byte(opts.ExpectedIdentityFingerprint)) != 1 {
			c.Close()
			return nil, fmt.Errorf("server identity fingerprint does not match pinned fingerprint")
		}
	}

	c.serverInfo = info
	if info.AuthRequired && apiKey == "" {
		c.Close()
		return nil, fmt.Errorf("server requires API key authentication")
	}

	if apiKey != "" {
		var out any
		if err = c.request(ctx, protocol.OpAuth, map[string]any{
			"api_key": apiKey,
		}, &out, nil); err != nil {
			c.Close()
			return nil, err
		}
	}

	return c, nil
}

// ServerInfo returns metadata and the verified dial-time identity, if configured.
// ServerInfo: Provides the Server Info operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) ServerInfo() ServerInfo {
	return c.serverInfo
}

// Close: Release transport resources; outstanding requests or subscriptions may stop receiving replies after closure.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closed)
		err = c.conn.Close()
	})

	return err
}

// readLoop: Read Loop using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) readLoop() {
	r := bufio.NewReader(c.conn)
	defer func() {
		c.closeOnce.Do(func() {
			close(c.closed)
			_ = c.conn.Close()
		})
		c.pendingMu.Lock()

		for id, ch := range c.pending {
			close(ch)
			delete(c.pending, id)
		}

		c.pendingMu.Unlock()
		c.subsMu.Lock()

		for id, ch := range c.subs {
			close(ch)
			delete(c.subs, id)
		}

		clear(c.subTargets)
		c.subsMu.Unlock()
	}()

	for {
		f, err := protocol.ReadBufferedFrame(r)
		if err != nil {
			return
		}

		if f.Kind == protocol.KindEvent {
			var m PubSubMessage
			if json.Unmarshal(f.Payload, &m) == nil {
				c.subsMu.Lock()
				ch := c.subs[f.RequestID]

				if ch != nil {
					select {
					case ch <- m:
					default:
					}
				}
				c.subsMu.Unlock()
			}
			continue
		}
		c.pendingMu.Lock()

		ch := c.pending[f.RequestID]
		if ch != nil {
			delete(c.pending, f.RequestID)
			ch <- f

			close(ch)
		}

		c.pendingMu.Unlock()
	}
}

// RequestError exposes the stable server error code without parsing message strings.
// RequestError: Represents an Amaquet request error with diagnostic information for callers; avoid logging credentials or private payloads.
// Manages native Amaquet TCP/TLS connections, request identifiers, authentication, backpressure, pending replies, subscriptions, and cleanup.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type RequestError struct {
	// Code is the server-defined machine-readable error code.
	Code string
	// Message is the server-defined human-readable error message.
	Message string
}

// Error: Represents an Amaquet error with diagnostic information for callers; avoid logging credentials or private payloads.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (e *RequestError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Code + ": " + e.Message
}

// Is: Provides the Is operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (e *RequestError) Is(target error) bool {
	t, ok := target.(*RequestError)
	return ok && (t.Code == "" || e.Code == t.Code)
}

// writeFrame bounds network writes even when the peer stops reading.
// writeFrame: Write Frame using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) writeFrame(ctx context.Context, f protocol.Frame) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case <-c.closed:
		return net.ErrClosed
	default:
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	limit := c.Timeout
	if limit <= 0 {
		limit = 30 * time.Second
	}

	deadline := time.Now().Add(limit)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}

	if err := c.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}

	err := protocol.WriteFrame(c.conn, f)
	_ = c.conn.SetWriteDeadline(time.Time{})

	return err
}

// sendCancel: Send Cancel using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) sendCancel(target uint64) {
	id := c.next.Add(1)
	b, _ := json.Marshal(map[string]any{"request_id": target})

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_ = c.writeFrame(ctx, protocol.Frame{
			Kind:      protocol.KindRequest,
			Opcode:    protocol.OpCancel,
			RequestID: id,
			Payload:   b,
		})
	}()
}

// request: Submit an operation with a request ID and await its matching response or a timeout or connection error.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `out`: Destination to decode the server response; preserve numeric precision.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) request(ctx context.Context, op protocol.Opcode, payload any, out any, eventCh chan PubSubMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	select {
	case <-c.closed:
		return net.ErrClosed

	default:
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	id := c.next.Add(1)
	ch := make(chan protocol.Frame, 1)

	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	if eventCh != nil {
		c.subsMu.Lock()
		c.subs[id] = eventCh
		c.subsMu.Unlock()
	}

	err = c.writeFrame(ctx, protocol.Frame{
		Kind:      protocol.KindRequest,
		Opcode:    op,
		RequestID: id,
		Payload:   b,
	})

	if err != nil {
		_ = c.Close() // cannot safely reuse a stream after a partial frame write
		c.pendingMu.Lock()
		delete(c.pending, id)

		c.pendingMu.Unlock()
		if eventCh != nil {
			c.subsMu.Lock()
			delete(c.subs, id)
			c.subsMu.Unlock()
		}

		return err
	}

	wait := c.Timeout
	if wait <= 0 {
		wait = 30 * time.Second
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		c.sendCancel(id)
		if eventCh != nil {
			c.subsMu.Lock()
			delete(c.subs, id)
			c.subsMu.Unlock()
		}
		return ctx.Err()

	case <-timer.C:
		c.sendCancel(id)
		if eventCh != nil {
			c.subsMu.Lock()
			delete(c.subs, id)
			c.subsMu.Unlock()
		}
		return context.DeadlineExceeded

	case <-c.closed:
		if eventCh != nil {
			c.subsMu.Lock()
			delete(c.subs, id)
			c.subsMu.Unlock()
		}
		return net.ErrClosed

	case f, ok := <-ch:
		if !ok {
			return net.ErrClosed
		}

		var resp struct {
			OK     bool            `json:"ok"`
			Result json.RawMessage `json:"result"`
			Error  *protocol.Error `json:"error"`
		}

		if err = json.Unmarshal(f.Payload, &resp); err != nil {
			return err
		}

		if !resp.OK {
			if eventCh != nil {
				c.subsMu.Lock()
				delete(c.subs, id)

				c.subsMu.Unlock()
			}

			if resp.Error == nil {
				return fmt.Errorf("request failed")
			}

			return &RequestError{Code: resp.Error.Code, Message: resp.Error.Message}
		}

		if out != nil {
			if raw, ok := out.(*json.RawMessage); ok {
				*raw = append((*raw)[:0], resp.Result...)
				return nil
			}

			return decodeNumber(resp.Result, out)
		}

		return nil
	}
}

// Command: Send a named native command with its JSON payload; this is the extension point for supported server commands.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `args`: Exact JSON argument properties supported by the server; names are not translated.
// Parameter `out`: Destination to decode the server response; preserve numeric precision.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Command(ctx context.Context, command string, args map[string]any, out any) error {
	return c.request(ctx, protocol.OpCommand, map[string]any{
		"command": command,
		"args":    args,
	}, out, nil)
}

// Hello: Provides the Hello operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Hello(ctx context.Context, nonce string) (ServerInfo, error) {
	var out ServerInfo
	err := c.request(ctx, protocol.OpHello, map[string]any{
		"nonce":        nonce,
		"min_protocol": int(protocol.Version),
		"max_protocol": int(protocol.Version),
	}, &out, nil)

	return out, err
}

// computedFingerprint derives the pin from verified public key bytes, not from
// an untrusted fingerprint string supplied by the network peer.
// computedFingerprint: Provides the computed Fingerprint operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (i ServerInfo) computedFingerprint() (string, bool) {
	pub, err := base64.RawStdEncoding.DecodeString(i.IdentityPublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return "", false
	}

	fingerprint := sha256.Sum256(pub)
	return "ed25519:" + hex.EncodeToString(fingerprint[:]), true
}

// VerifyNonce: Verify Nonce using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (i ServerInfo) VerifyNonce(nonce string) bool {
	if i.IdentityPublicKey == "" || i.NonceSignature == "" {
		return false
	}

	pub, e := base64.RawStdEncoding.DecodeString(i.IdentityPublicKey)
	if e != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}

	fingerprint, ok := i.computedFingerprint()
	if !ok || (i.IdentityFingerprint != "" && subtle.ConstantTimeCompare(
		[]byte(i.IdentityFingerprint),
		[]byte(fingerprint),
	) != 1) {
		return false
	}

	sig, e := base64.RawStdEncoding.DecodeString(i.NonceSignature)
	return e == nil && ed25519.Verify(ed25519.PublicKey(pub), []byte(nonce), sig)
}

// Ping: Send a liveness probe. A successful response confirms the endpoint accepts this command.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Ping(ctx context.Context) error {
	var out string
	return c.Command(ctx, "PING", nil, &out)
}

// Set: Write a value using the server documented type and options; this may replace existing state.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Parameter `value`: Data to write or transmit, using the wire-type JSON format.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Set(ctx context.Context, key, typeName string, value any, ttl time.Duration) error {
	return c.Command(ctx, "SET", map[string]any{
		"key":    key,
		"type":   typeName,
		"value":  value,
		"ttl_ms": ttl.Milliseconds(),
	}, nil)
}

// Get: Read current data for a key or type-specific field, subject to server permissions and key existence.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Get(ctx context.Context, key string) (map[string]any, error) {
	var out map[string]any
	err := c.Command(ctx, "GET", map[string]any{"key": key}, &out)

	return out, err
}

// Delete: Remove one or more keys or named items as defined by the command arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Delete(ctx context.Context, keys ...string) (int, error) {
	var n int
	err := c.Command(ctx, "DEL", map[string]any{"keys": keys}, &n)

	return n, err
}

// Create: Create a new value with a registered wire type and its construction options.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Parameter `options`: Call settings or documented construction flags, as accepted by this method.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Create(ctx context.Context, key, typeName string, options map[string]any) error {
	return c.Command(ctx, "CREATE", map[string]any{
		"key":     key,
		"type":    typeName,
		"options": options,
	}, nil)
}

// Op: Dispatch a typed OP command using the key and its registered wire type.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Parameter `args`: Exact JSON argument properties supported by the server; names are not translated.
// Parameter `out`: Destination to decode the server response; preserve numeric precision.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Op(ctx context.Context, key, operation string, args map[string]any, out any) error {
	return c.Command(ctx, "OP", map[string]any{
		"key":       key,
		"operation": operation,
		"args":      args,
	}, out)
}

// tryUnsubscribe is needed because CANCEL only cancels commands, not SUBSCRIBE.
// tryUnsubscribe: Provides the try Unsubscribe operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) tryUnsubscribe(key, channel string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_ = c.request(ctx, protocol.OpUnsubscribe, map[string]any{
			"key":     key,
			"channel": channel,
		}, nil, nil)
	}()
}

// Subscribe: Register a channel subscription; consume events from the bounded local queue and unsubscribe during cleanup.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Parameter `buffer`: Bounded storage for incoming payloads or events.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Subscribe(ctx context.Context, key, channel string, buffer int) (*Subscription, error) {
	if buffer < 1 {
		buffer = 64
	}

	events := make(chan PubSubMessage, buffer)
	id := c.next.Add(1)
	ch := make(chan protocol.Frame, 1)

	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	target := key + "\x00" + channel
	c.subsMu.Lock()

	if _, exists := c.subTargets[target]; exists {
		c.subsMu.Unlock()
		c.pendingMu.Lock()
		delete(c.pending, id)

		c.pendingMu.Unlock()
		return nil, fmt.Errorf("already subscribed to key %q channel %q on this connection", key, channel)
	}

	if c.subTargets == nil {
		c.subTargets = map[string]uint64{}
	}

	c.subTargets[target] = id
	c.subs[id] = events
	c.subsMu.Unlock()

	cleanup := func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()

		c.subsMu.Lock()
		delete(c.subTargets, target)

		if x := c.subs[id]; x != nil {
			delete(c.subs, id)
			close(x)
		}

		c.subsMu.Unlock()
	}

	b, _ := json.Marshal(map[string]any{
		"key":     key,
		"channel": channel,
		"buffer":  buffer,
	})

	err := c.writeFrame(ctx, protocol.Frame{
		Kind:      protocol.KindRequest,
		Opcode:    protocol.OpSubscribe,
		RequestID: id,
		Payload:   b,
	})

	if err != nil {
		cleanup()
		return nil, err
	}

	wait := c.Timeout
	if wait <= 0 {
		wait = 30 * time.Second
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-timer.C:
		cleanup()
		c.tryUnsubscribe(key, channel)
		return nil, context.DeadlineExceeded

	case <-ctx.Done():
		cleanup()
		c.tryUnsubscribe(key, channel)
		return nil, ctx.Err()

	case <-c.closed:
		cleanup()
		return nil, net.ErrClosed

	case f, ok := <-ch:
		if !ok {
			cleanup()
			return nil, net.ErrClosed
		}

		var resp protocol.Response
		if e := json.Unmarshal(f.Payload, &resp); e != nil {
			cleanup()
			return nil, e
		}

		if !resp.OK {
			cleanup()
			if resp.Error != nil {
				return nil, &RequestError{Code: resp.Error.Code, Message: resp.Error.Message}
			}

			return nil, fmt.Errorf("subscribe failed without error details")
		}
	}

	sub := &Subscription{C: events}

	var once sync.Once
	var unsubscribeErr error

	sub.close = func() error {
		once.Do(func() {
			var out any
			err := c.request(context.Background(), protocol.OpUnsubscribe, map[string]any{
				"key":     key,
				"channel": channel,
			}, &out, nil)
			cleanup()
			unsubscribeErr = err
		})
		return unsubscribeErr
	}

	return sub, nil
}

// BlobRead: Provides the Blob Read operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) BlobRead(ctx context.Context, key string, offset int64, length int) ([]byte, int64, error) {
	rng, err := c.BlobReadRange(ctx, key, offset, length)
	return rng.Data, rng.Size, err
}

// UploadBlob is the compatibility helper for streaming a binary value.
// UploadBlob: Provides the Upload Blob operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) UploadBlob(ctx context.Context, key, uploadID string, r io.Reader, size int64, chunkSize int) error {
	return c.UploadBlobWithOptions(ctx, key, r, size, BlobOptions{
		UploadID:  uploadID,
		ChunkSize: chunkSize,
	})
}
