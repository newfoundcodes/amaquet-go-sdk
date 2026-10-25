package amaquet

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"

	"github.com/newfoundcodes/amaquet-go-sdk/internal/protocol"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// FuzzSDKNestedValueRoundTrip exercises typed value envelopes and JSON number preservation.
func FuzzSDKNestedValueRoundTrip(f *testing.F) {
	f.Add("integer", "9223372036854775807")
	f.Add("json", "18446744073709551615")
	f.Add("utf8_string", "\x00\u2603")
	f.Fuzz(func(t *testing.T, typ, raw string) {
		if len(typ) > 512 || len(raw) > 1024 {
			return
		}
		value := Typed(typ, raw)
		b, e := json.Marshal(value)
		if e != nil {
			t.Fatal(e)
		}
		var v Value
		if e = json.Unmarshal(b, &v); e != nil {
			t.Fatal(e)
		}
		if v.Type != typ || v.Value != raw {
			t.Fatalf("changed typed value: %#v", v)
		}
	})
}
func FuzzSDKWriteOptions(f *testing.F) {
	f.Add(int64(1000), true, false, int64(0))
	f.Add(int64(-1), false, true, int64(0))
	f.Add(int64(0), true, true, int64(0))
	f.Add(int64(0), false, false, int64(1740000000))
	f.Fuzz(func(t *testing.T, ms int64, nx, xx bool, ts int64) {
		if ms < -10_000_000 || ms > 10_000_000 {
			return
		}
		opts := WriteOptions{TTL: time.Duration(ms) * time.Millisecond, NX: nx, XX: xx}
		if ts > 0 && ts < 2147483647 {
			opts.ExpiresAt = time.Unix(ts, 0)
		}
		args := map[string]any{}
		err := opts.apply(args)
		if nx && xx && err == nil {
			t.Fatal("accepted mutually exclusive flags")
		}
		if ms < 0 && err == nil {
			t.Fatal("accepted negative TTL")
		}
		if err == nil {
			b, e := json.Marshal(args)
			if e != nil || !json.Valid(b) {
				t.Fatalf("malformed options: %s %v", b, e)
			}
		}
	})
}
func FuzzSDKAdminResourceIDs(f *testing.F) {
	for _, s := range []string{"abc", "..", "a/b", "a%2fb", "some üñïçøde", "?", "#", "x\x00y"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, id string) {
		if len(id) > 512 {
			return
		}
		escaped, err := adminID(id)
		if err != nil {
			return
		}
		decoded, err := url.PathUnescape(escaped)
		if err != nil || decoded != id {
			t.Fatalf("round-trip path ID: %q->%q->%q: %v", id, escaped, decoded, err)
		}
		if strings.ContainsAny(decoded, "/\\\x00") {
			t.Fatalf("unsafe path id %q", id)
		}
	})
}
func FuzzSDKAdminUntrustedPath(f *testing.F) {
	f.Add("/api/health")
	f.Add("//evil.example/steal")
	f.Add("/../../etc/passwd")
	f.Add("/api/data/key?key=a%26b")
	f.Fuzz(func(t *testing.T, path string) {
		if len(path) > 1024 {
			return
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"ok":true}`)) }))
		defer server.Close()
		c, err := NewAdminClient(server.URL, AdminOptions{})
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]any
		err = c.RequestJSON(context.Background(), "GET", path, nil, &result)
		if errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	})
}
func FuzzSDKExactJSONNumbers(f *testing.F) {
	f.Add("18446744073709551615")
	f.Add("-9223372036854775808")
	f.Add("1.00000000000000000000000000000001")
	f.Fuzz(func(t *testing.T, number string) {
		if len(number) > 1024 || !json.Valid([]byte(number)) {
			return
		}
		var x any
		if err := decodeNumber([]byte(number), &x); err != nil {
			return
		}
		if _, ok := x.(json.Number); ok {
			if strings.TrimSpace(number) != x.(json.Number).String() {
				t.Fatal("numeric precision loss")
			}
		}
	})
}
func FuzzSDKBlobChunkBounds(f *testing.F) {
	f.Add(int64(0), []byte{1, 2, 3})
	f.Add(int64(-1), []byte{1})
	f.Add(int64(0), []byte{})
	f.Fuzz(func(t *testing.T, offset int64, data []byte) {
		if len(data) > 2048 {
			return
		}
		if len(data) > 0 && offset >= 0 {
			return
		}
		c := &Client{}
		_, err := c.BlobChunk(context.Background(), "id", offset, data)
		if (len(data) == 0 || offset < 0) && err == nil {
			t.Fatalf("invalid blob chunk accepted")
		}
	})
}
func TestSDKDecodeNumberRoundTrip(t *testing.T) {
	b := []byte(`{"x":18446744073709551615,"y":0.12345678901234567890123456789}`)
	var m map[string]any
	if err := decodeNumber(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["x"].(json.Number).String() != "18446744073709551615" {
		t.Fatal(m["x"])
	}
	if !bytes.Contains([]byte(m["y"].(json.Number)), []byte("12345678901234567890123456789")) {
		t.Fatal(m["y"])
	}
}

// FuzzSDKNativeFrameRoundTrip checks all header fields and opaque payload bytes.
func FuzzSDKNativeFrameRoundTrip(f *testing.F) {
	f.Add(uint8(1), uint16(0), uint16(1), uint64(1), []byte(`{"command":"PING"}`))
	f.Add(uint8(3), uint16(65535), uint16(65535), ^uint64(0), []byte{0, 255, 0, 3})
	f.Fuzz(func(t *testing.T, kind uint8, flags, opcode uint16, id uint64, payload []byte) {
		if len(payload) > 4096 {
			return
		}
		input := protocol.Frame{Kind: protocol.Kind(kind), Flags: flags, Opcode: protocol.Opcode(opcode), RequestID: id, Payload: payload}
		var buf bytes.Buffer
		if err := protocol.WriteFrame(&buf, input); err != nil {
			t.Fatal(err)
		}
		got, err := protocol.ReadFrameLimit(bytes.NewReader(buf.Bytes()), 4096)
		if err != nil {
			t.Fatal(err)
		}
		if got.Kind != input.Kind || got.Flags != input.Flags || got.Opcode != input.Opcode || got.RequestID != input.RequestID || !bytes.Equal(got.Payload, input.Payload) {
			t.Fatalf("protocol round-trip mismatch: %#v != %#v", got, input)
		}
	})
}

// FuzzSDKMalformedNativeFrame ensures malformed and truncated wire data is rejected safely.
func FuzzSDKMalformedNativeFrame(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("AMQT"))
	f.Add(bytes.Repeat([]byte{0xff}, protocol.HeaderSize))
	f.Add([]byte("AMQT\x01\x01\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\xff\xff\xff\xff"))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 4096 {
			return
		}
		_, _ = protocol.ReadFrameLimit(bytes.NewReader(input), 4096)
	})
}

// FuzzSDKEndpointParsing checks URI parsing and address construction.
func FuzzSDKEndpointParsing(f *testing.F) {
	for _, s := range []string{"amaquet://localhost", "amaquets://[::1]:13378", "amaquet://127.0.0.1:0", "amaquet://host/db", "http://host", "amaquet://u:p@host:5", "amaquet://host?api_key=a%26b"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 1024 {
			return
		}
		ep, err := protocol.ParseEndpoint(raw)
		if err != nil {
			return
		}
		if ep.Host == "" || ep.Port < 1 || ep.Port > 65535 || ep.Database != "" {
			t.Fatalf("accepted invalid parsed endpoint: %#v", ep)
		}
		if _, _, err := net.SplitHostPort(ep.Address()); err != nil {
			t.Fatalf("invalid endpoint address %q: %v", ep.Address(), err)
		}
	})
}

// FuzzSDKCommandJSON checks untrusted command payload decoding and number preservation.
func FuzzSDKCommandJSON(f *testing.F) {
	f.Add([]byte(`{"command":"SET","args":{"key":"a","value":18446744073709551615}}`))
	f.Add([]byte(`{"command":"OP","args":{"key":"a","args":{"nested":[1,true,null]}}}`))
	f.Add([]byte("not json"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 8192 {
			return
		}
		req, err := protocol.DecodeRequest(raw)
		if err != nil {
			return
		}
		for key, value := range req.Args {
			if !json.Valid(value) {
				t.Fatalf("decoded invalid JSON for %q: %s", key, value)
			}
		}
	})
}

// FuzzSDKTypeRegistry validates stable mapping between type names and documented methods.
func FuzzSDKTypeRegistry(f *testing.F) {
	for _, name := range []string{"list", "hnsw_index", "reliable_queue", "json", "undefined", "", "a/../../x"} {
		f.Add(name)
	}
	f.Fuzz(func(t *testing.T, name string) {
		if len(name) > 1024 {
			return
		}
		ops := DocumentedOperations(name)
		seen := map[string]bool{}
		for _, op := range ops {
			if op == "" || seen[op] {
				t.Fatalf("invalid or duplicate type-specific operation %q: %q", name, op)
			}
			seen[op] = true
		}
	})
}

// FuzzSDKNonceVerification confirms no other challenge passes a signed identity.
func FuzzSDKNonceVerification(f *testing.F) {
	f.Add("challenge", "different")
	f.Add("", "changed")
	f.Add("same", "same")
	f.Fuzz(func(t *testing.T, signed, other string) {
		if len(signed) > 1024 || len(other) > 1024 {
			return
		}
		seed := [32]byte{1, 2, 3, 4}
		private := ed25519.NewKeyFromSeed(seed[:])
		info := ServerInfo{
			IdentityPublicKey: base64.RawStdEncoding.EncodeToString(private.Public().(ed25519.PublicKey)),
			NonceSignature:    base64.RawStdEncoding.EncodeToString(ed25519.Sign(private, []byte(signed))),
		}
		if info.VerifyNonce(other) != (signed == other) {
			t.Fatalf("unexpected nonce verification result")
		}
	})
}

// FuzzSDKBlobRangeArguments checks invalid offset/length pairs before network I/O.
func FuzzSDKBlobRangeArguments(f *testing.F) {
	f.Add(int64(-1), 1)
	f.Add(int64(0), -1)
	f.Add(int64(0), MaxBlobReadSize+1)
	f.Fuzz(func(t *testing.T, offset int64, length int) {
		if offset >= 0 && length >= 0 && length <= MaxBlobReadSize {
			return
		}
		var c Client
		_, err := c.BlobReadRange(context.Background(), "obj", offset, length)
		if err == nil {
			t.Fatalf("accepted invalid range: %d %d", offset, length)
		}
	})
}
