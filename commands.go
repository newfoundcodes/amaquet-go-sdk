package amaquet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Value is the explicit envelope required for values nested in Amaquet collections.
// Use json.Number, string, or integer types for exact numeric representation.
// Value: Provides the Value operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Provides command-level helpers using the exact names and JSON field contracts used by the Amaquet native dispatcher.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type Value struct {
	// Type is the Amaquet wire type name of the nested value.
	Type string `json:"type"`
	// Value is the JSON-compatible value encoded inside the envelope.
	Value any `json:"value"`
}

// Typed: Provides the Typed operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `value`: Data to write or transmit, using the wire-type JSON format.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func Typed(typeName string, value any) Value {
	return Value{Type: typeName, Value: value}
}

// KeySnapshot is the wire snapshot returned by GET. RawMessage preserves large JSON
// numbers and avoids converting int64, uint64, or exact decimals into float64.
// KeySnapshot: Provides the Key Snapshot operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Provides command-level helpers using the exact names and JSON field contracts used by the Amaquet native dispatcher.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type KeySnapshot struct {
	// Key is the key associated with the snapshot.
	Key string `json:"key"`
	// Type is the Amaquet wire type stored at Key.
	Type string `json:"type"`
	// Value is the raw JSON representation of the stored value.
	Value json.RawMessage `json:"value"`
	// Version is the server-side version of the key.
	Version uint64 `json:"version"`
	// ExpiresAt is the absolute expiration time, when one is configured.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Decode: Decode using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `out`: Destination to decode the server response; preserve numeric precision.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (s KeySnapshot) Decode(out any) error {
	return decodeNumber(s.Value, out)
}

// decodeNumber: Decode Number using the implementation's validation, error, and resource-lifetime rules.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `out`: Destination to decode the server response; preserve numeric precision.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func decodeNumber(data []byte, out any) error {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()

	if err := dec.Decode(out); err != nil {
		return err
	}

	return nil
}

// WriteOptions are sent as native wire fields, not interpreted client-side.
// WriteOptions: Write Options using the implementation's validation, error, and resource-lifetime rules.
// Provides command-level helpers using the exact names and JSON field contracts used by the Amaquet native dispatcher.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type WriteOptions struct {
	// TTL requests a relative expiration duration.
	TTL time.Duration
	// NX requests creation only when the key does not already exist.
	NX bool
	// XX requests writing only when the key already exists.
	XX bool
	// ExpiresAt requests an absolute expiration time.
	ExpiresAt time.Time
}

// apply: Provides the apply operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `args`: Exact JSON argument properties supported by the server; names are not translated.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (o WriteOptions) apply(args map[string]any) error {
	if o.NX && o.XX {
		return errors.New("nx and xx cannot both be true")
	}

	if o.TTL < 0 {
		return errors.New("ttl cannot be negative")
	}

	if o.TTL > 0 && o.TTL < time.Millisecond {
		return errors.New("ttl must be at least 1 millisecond")
	}

	if !o.ExpiresAt.IsZero() && o.ExpiresAt.UnixNano() <= 0 {
		return errors.New("absolute expiry must be after the Unix epoch")
	}

	if o.TTL > 0 && !o.ExpiresAt.IsZero() {
		return errors.New("ttl and expires_at are mutually exclusive")
	}

	if o.TTL > 0 {
		args["ttl_ms"] = o.TTL.Milliseconds()
	}

	if o.NX {
		args["nx"] = true
	}

	if o.XX {
		args["xx"] = true
	}

	if !o.ExpiresAt.IsZero() {
		args["expires_at_unix_nano"] = o.ExpiresAt.UnixNano()
	}

	return nil
}

// SetWithOptions: Provides the Set With Options operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Parameter `value`: Data to write or transmit, using the wire-type JSON format.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) SetWithOptions(ctx context.Context, key, typ string, value any, opts WriteOptions) error {
	a := map[string]any{"key": key, "type": typ, "value": value}
	if err := opts.apply(a); err != nil {
		return err
	}

	return c.Command(ctx, "SET", a, nil)
}

// CreateWithOptions: Provides the Create With Options operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Parameter `options`: Call settings or documented construction flags, as accepted by this method.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) CreateWithOptions(ctx context.Context, key, typ string, options map[string]any, opts WriteOptions) error {
	a := map[string]any{"key": key, "type": typ, "options": options}
	if err := opts.apply(a); err != nil {
		return err
	}

	return c.Command(ctx, "CREATE", a, nil)
}

// CompareAndSet: Provides the Compare And Set operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Parameter `value`: Data to write or transmit, using the wire-type JSON format.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) CompareAndSet(ctx context.Context, key string, version uint64, typ string, value any, opts WriteOptions) error {
	if version == 0 {
		return errors.New("expected_version must be positive")
	}

	if opts.NX || opts.XX {
		return errors.New("CAS does not support nx or xx")
	}

	a := map[string]any{
		"key":              key,
		"expected_version": version,
		"type":             typ,
		"value":            value,
	}

	if err := opts.apply(a); err != nil {
		return err
	}

	return c.Command(ctx, "CAS", a, nil)
}

// GetSnapshot: Provides the Get Snapshot operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) GetSnapshot(ctx context.Context, key string) (KeySnapshot, error) {
	var s KeySnapshot
	err := c.Command(ctx, "GET", map[string]any{"key": key}, &s)

	return s, err
}

// Exists: Provides the Exists operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	var v bool
	err := c.Command(ctx, "EXISTS", map[string]any{"key": key}, &v)
	return v, err
}

// TypeOf: Provides the Type Of operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) TypeOf(ctx context.Context, key string) (string, error) {
	var v string
	err := c.Command(ctx, "TYPE", map[string]any{"key": key}, &v)
	return v, err
}

// TTL: Provides the TTL operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) TTL(ctx context.Context, key string) (int64, error) {
	var v int64
	err := c.Command(ctx, "TTL", map[string]any{"key": key}, &v)
	return v, err
}

// Expire: Provides the Expire operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("ttl must be positive")
	}

	return c.Command(ctx, "EXPIRE", map[string]any{
		"key":    key,
		"ttl_ms": ttl.Milliseconds(),
	}, nil)
}

// ExpireAt: Provides the Expire At operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) ExpireAt(ctx context.Context, key string, at time.Time) error {
	if at.IsZero() {
		return errors.New("expiry must not be zero")
	}

	return c.Command(ctx, "EXPIRE", map[string]any{
		"key":                  key,
		"expires_at_unix_nano": at.UnixNano(),
	}, nil)
}

// Persist: Provides the Persist operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Persist(ctx context.Context, key string) error {
	return c.Command(ctx, "PERSIST", map[string]any{"key": key}, nil)
}

// Keys: Provides the Keys operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Keys(ctx context.Context, prefix string, limit int) ([]string, error) {
	var v []string
	err := c.Command(ctx, "KEYS", map[string]any{"prefix": prefix, "limit": limit}, &v)
	return v, err
}

// ScanPage: Provides the Scan Page operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Provides command-level helpers using the exact names and JSON field contracts used by the Amaquet native dispatcher.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type ScanPage struct {
	// Keys contains the keys returned for this page.
	Keys []string `json:"keys"`
	// Cursor identifies the next page; an empty cursor means scanning is complete.
	Cursor string `json:"cursor"`
}

// Scan: Provides the Scan operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Scan(ctx context.Context, cursor, prefix string, count int) (ScanPage, error) {
	var v ScanPage
	err := c.Command(ctx, "SCAN", map[string]any{
		"cursor": cursor,
		"prefix": prefix,
		"count":  count,
	}, &v)

	return v, err
}

// ScanAll: Provides the Scan All operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) ScanAll(ctx context.Context, prefix string, count int, visit func(string) error) error {
	if visit == nil {
		return errors.New("visit callback is required")
	}

	cursor := ""
	for {
		page, err := c.Scan(ctx, cursor, prefix, count)
		if err != nil {
			return err
		}

		for _, k := range page.Keys {
			if err := ctx.Err(); err != nil {
				return err
			}

			if err := visit(k); err != nil {
				return err
			}
		}

		if page.Cursor == "" {
			return nil
		}

		if page.Cursor == cursor {
			return fmt.Errorf("scan did not advance: %q", cursor)
		}

		cursor = page.Cursor
	}
}

// Types: Provides the Types operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Types(ctx context.Context) ([]string, error) {
	var v []string
	err := c.Command(ctx, "TYPES", nil, &v)
	return v, err
}

// Info: Provides the Info operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Info(ctx context.Context) (json.RawMessage, error) {
	return c.RawCommand(ctx, "INFO", nil)
}

// Memory: Provides the Memory operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Memory(ctx context.Context, key string) (json.RawMessage, error) {
	return c.RawCommand(ctx, "MEMORY", map[string]any{"key": key})
}

// RawCommand: Provides the Raw Command operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `args`: Exact JSON argument properties supported by the server; names are not translated.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) RawCommand(ctx context.Context, command string, args map[string]any) (json.RawMessage, error) {
	var raw json.RawMessage
	err := c.Command(ctx, command, args, &raw)
	return raw, err
}

// RawOp: Provides the Raw Op operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Parameter `args`: Exact JSON argument properties supported by the server; names are not translated.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) RawOp(ctx context.Context, key, op string, args map[string]any) (json.RawMessage, error) {
	var raw json.RawMessage
	err := c.Op(ctx, key, op, args, &raw)
	return raw, err
}

// BatchRequest mirrors the actual request schema; BATCH is sequential, not atomic.
// BatchRequest: Provides the Batch Request operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Provides command-level helpers using the exact names and JSON field contracts used by the Amaquet native dispatcher.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type BatchRequest struct {
	// Command is the native command to execute.
	Command string `json:"command"`
	// Args contains the command's JSON arguments.
	Args map[string]any `json:"args,omitempty"`
}

// BatchItem: Provides the Batch Item operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Provides command-level helpers using the exact names and JSON field contracts used by the Amaquet native dispatcher.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type BatchItem struct {
	// OK reports whether the individual command succeeded.
	OK bool `json:"ok"`
	// Result contains the raw JSON command result when the command succeeds.
	Result json.RawMessage `json:"result,omitempty"`
	// Error contains the server-reported error when the command fails.
	Error string `json:"error,omitempty"`
}

// Batch: Provides the Batch operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) Batch(ctx context.Context, requests []BatchRequest, continueOnError bool) ([]BatchItem, error) {
	if len(requests) == 0 || len(requests) > 1024 {
		return nil, errors.New("batch must contain 1 to 1024 requests")
	}

	var out []BatchItem
	err := c.Command(ctx, "BATCH", map[string]any{
		"requests":          requests,
		"continue_on_error": continueOnError,
	}, &out)

	return out, err
}

// Operation invokes a type-specific operation without altering argument field names.
// Methods generated for the 91 data types use this object under the hood.
// Operation: Dispatch a typed OP command using the key, exact uppercase operation name, and server-defined argument map.
// Provides command-level helpers using the exact names and JSON field contracts used by the Amaquet native dispatcher.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type Operation struct {
	client   *Client
	key      string
	typeName string
}

// Key: Provides the Key operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (o Operation) Key() string {
	return o.key
}

// TypeName: Provides the Type Name operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (o Operation) TypeName() string {
	return o.typeName
}

// Do: Provides the Do operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `args`: Exact JSON argument properties supported by the server; names are not translated.
// Parameter `out`: Destination to decode the server response; preserve numeric precision.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (o Operation) Do(ctx context.Context, op string, args map[string]any, out any) error {
	if o.client == nil {
		return netErrorNilClient()
	}

	return o.client.Op(ctx, o.key, op, args, out)
}

// Raw: Provides the Raw operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `args`: Exact JSON argument properties supported by the server; names are not translated.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (o Operation) Raw(ctx context.Context, op string, args map[string]any) (json.RawMessage, error) {
	if o.client == nil {
		return nil, netErrorNilClient()
	}

	return o.client.RawOp(ctx, o.key, op, args)
}

// Create: Create a new value with a registered wire type and its construction options.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `options`: Call settings or documented construction flags, as accepted by this method.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (o Operation) Create(ctx context.Context, options map[string]any, write WriteOptions) error {
	if o.client == nil {
		return netErrorNilClient()
	}

	return o.client.CreateWithOptions(ctx, o.key, o.typeName, options, write)
}

// Set: Write a value using the server documented type and options; this may replace existing state.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `value`: Data to write or transmit, using the wire-type JSON format.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (o Operation) Set(ctx context.Context, value any, write WriteOptions) error {
	if o.client == nil {
		return netErrorNilClient()
	}

	return o.client.SetWithOptions(ctx, o.key, o.typeName, value, write)
}

// Get: Read current data for a key or type-specific field, subject to server permissions and key existence.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (o Operation) Get(ctx context.Context) (KeySnapshot, error) {
	if o.client == nil {
		return KeySnapshot{}, netErrorNilClient()
	}

	return o.client.GetSnapshot(ctx, o.key)
}

// netErrorNilClient: Provides the net Error Nil Client operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func netErrorNilClient() error {
	return errors.New("amaquet: nil client")
}

// ForType: Provides the For Type operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) ForType(key, typ string) Operation {
	return Operation{client: c, key: key, typeName: typ}
}

// IsCode reports whether err is a native Amaquet protocol failure with code.
// IsCode: Provides the Is Code operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func IsCode(err error, code string) bool {
	return errors.Is(err, &RequestError{Code: code})
}
