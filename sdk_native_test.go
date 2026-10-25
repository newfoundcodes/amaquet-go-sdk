package amaquet

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/newfoundcodes/amaquet-go-sdk/internal/protocol"
)

type fakeNative struct {
	t        *testing.T
	listener net.Listener
	mu       sync.Mutex
	blobs    map[string][]byte
	active   map[string]int64
	private  ed25519.PrivateKey
	auth     bool
	close    sync.Once
}

func newFakeNative(t *testing.T, auth bool) *fakeNative {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return newFakeNativeListener(t, auth, ln)
}
func newFakeNativeListener(t *testing.T, auth bool, ln net.Listener) *fakeNative {
	t.Helper()
	_, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeNative{t: t, listener: ln, blobs: map[string][]byte{}, active: map[string]int64{}, private: sk, auth: auth}
	go func() {
		for {
			conn, e := ln.Accept()
			if e != nil {
				return
			}
			go f.handle(conn)
		}
	}()
	t.Cleanup(func() { f.close.Do(func() { ln.Close() }) })
	return f
}
func (f *fakeNative) url() string { return "amaquet://" + f.listener.Addr().String() }
func (f *fakeNative) handle(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	var writeMu sync.Mutex
	write := func(id uint64, result any, protoErr *protocol.Error) {
		resp := protocol.Response{OK: protoErr == nil, Result: result, Error: protoErr}
		b, _ := json.Marshal(resp)
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = protocol.WriteFrame(conn, protocol.Frame{Kind: protocol.KindResponse, RequestID: id, Payload: b})
	}
	for {
		fr, err := protocol.ReadBufferedFrame(r)
		if err != nil {
			return
		}
		switch fr.Opcode {
		case protocol.OpHello:
			var hello struct {
				Nonce string `json:"nonce"`
			}
			_ = json.Unmarshal(fr.Payload, &hello)
			pub := f.private.Public().(ed25519.PublicKey)
			write(fr.RequestID, ServerInfo{Name: "Amaquet", Protocol: 1, SelectedProtocol: 1, AuthRequired: f.auth, IdentityPublicKey: base64.RawStdEncoding.EncodeToString(pub), NonceSignature: base64.RawStdEncoding.EncodeToString(ed25519.Sign(f.private, []byte(hello.Nonce)))}, nil)
		case protocol.OpAuth:
			var v struct {
				APIKey string `json:"api_key"`
			}
			_ = json.Unmarshal(fr.Payload, &v)
			if v.APIKey != "test-key" {
				write(fr.RequestID, nil, &protocol.Error{Code: "UNAUTHORIZED", Message: "bad credential"})
			} else {
				write(fr.RequestID, map[string]any{"authenticated": true}, nil)
			}
		case protocol.OpSubscribe:
			var sub struct {
				Key     string `json:"key"`
				Channel string `json:"channel"`
			}
			_ = json.Unmarshal(fr.Payload, &sub)
			write(fr.RequestID, map[string]any{"subscribed": true}, nil)
			go func(id uint64, ch string) {
				time.Sleep(2 * time.Millisecond)
				payload, _ := json.Marshal(PubSubMessage{Channel: ch, Payload: "event-data", PublishedAt: time.Now().UTC()})
				writeMu.Lock()
				defer writeMu.Unlock()
				_ = protocol.WriteFrame(conn, protocol.Frame{Kind: protocol.KindEvent, RequestID: id, Payload: payload})
			}(fr.RequestID, sub.Channel)
		case protocol.OpCancel: // no response by server contract
		case protocol.OpCommand:
			var p protocol.Request
			_ = json.Unmarshal(fr.Payload, &p)
			args := map[string]json.RawMessage(p.Args)
			str := func(k string) string { var s string; _ = json.Unmarshal(args[k], &s); return s }
			num := func(k string) int64 { var n int64; _ = json.Unmarshal(args[k], &n); return n }
			switch p.Command {
			case "PING":
				write(fr.RequestID, "PONG", nil)
			case "ECHO_DELAY":
				var n int
				_ = json.Unmarshal(args["delay_ms"], &n)
				go func(id uint64) {
					time.Sleep(time.Duration(n) * time.Millisecond)
					write(id, map[string]any{"id": id}, nil)
				}(fr.RequestID)
			case "GET":
				write(fr.RequestID, map[string]any{"key": str("key"), "type": "unsigned_integer", "version": uint64(1), "value": json.Number("18446744073709551615")}, nil)
			case "EXISTS":
				write(fr.RequestID, true, nil)
			case "TYPE":
				write(fr.RequestID, "unsigned_integer", nil)
			case "TTL":
				write(fr.RequestID, int64(-1), nil)
			case "TYPES":
				write(fr.RequestID, RegisteredTypes(), nil)
			case "SCAN":
				write(fr.RequestID, ScanPage{Keys: []string{"a", "b"}}, nil)
			case "KEYS":
				write(fr.RequestID, []string{"a", "b"}, nil)
			case "BATCH":
				write(fr.RequestID, []BatchItem{{OK: true, Result: json.RawMessage("true")}}, nil)
			case "OP":
				if str("operation") == "FAIL" {
					write(fr.RequestID, nil, &protocol.Error{Code: "WRONG_TYPE", Message: "bad type"})
				} else {
					write(fr.RequestID, uint64(22), nil)
				}
			case "BLOB_BEGIN":
				id := str("upload_id")
				total := num("total_size")
				f.mu.Lock()
				f.active[id] = total
				f.blobs[id] = make([]byte, int(total))
				f.mu.Unlock()
				write(fr.RequestID, BlobUpload{UploadID: id, TotalSize: total}, nil)
			case "BLOB_CHUNK":
				id := str("upload_id")
				off := num("offset")
				var enc string
				_ = json.Unmarshal(args["data"], &enc)
				data, _ := base64.StdEncoding.DecodeString(enc)
				f.mu.Lock()
				b := f.blobs[id]
				copy(b[int(off):], data)
				f.blobs[id] = b
				f.mu.Unlock()
				write(fr.RequestID, map[string]any{"received": off + int64(len(data)), "total_size": int64(len(b))}, nil)
			case "BLOB_COMMIT", "BLOB_ABORT":
				write(fr.RequestID, true, nil)
			case "BLOB_READ":
				key := str("key")
				off := num("offset")
				n := num("length")
				f.mu.Lock()
				b := append([]byte(nil), f.blobs[key]...)
				f.mu.Unlock()
				if off > int64(len(b)) {
					write(fr.RequestID, nil, &protocol.Error{Code: "ERROR", Message: "bad offset"})
					break
				}
				end := off + n
				if end > int64(len(b)) {
					end = int64(len(b))
				}
				write(fr.RequestID, map[string]any{"key": key, "version": uint64(1), "offset": off, "size": len(b), "data": base64.StdEncoding.EncodeToString(b[int(off):int(end)])}, nil)
			default:
				write(fr.RequestID, true, nil)
			}
		default:
			write(fr.RequestID, true, nil)
		}
	}
}
func dialFake(t *testing.T, f *fakeNative) *Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	opts := DialOptions{Timeout: 2 * time.Second, RequireIdentity: true}
	if f.auth {
		opts.APIKey = "test-key"
	}
	c, err := DialWithOptions(ctx, f.url(), opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}
func TestSDKVerifiedTLSAndSignedServerIdentity(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, pub, priv)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(root)
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: priv}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = newFakeNativeListener(t, true, listener)
	uri := "amaquets://" + listener.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	client, err := DialWithOptions(ctx, uri, DialOptions{
		APIKey: "test-key", RequireIdentity: true, Timeout: 2 * time.Second,
		TLSConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if !client.ServerInfo().AuthRequired || client.ServerInfo().IdentityPublicKey == "" {
		t.Fatal("authenticated TLS peer did not return signed identity")
	}
	_, err = DialWithOptions(ctx, uri, DialOptions{APIKey: "test-key", Timeout: time.Second})
	if err == nil {
		t.Fatal("accepted untrusted server certificate")
	}
}

func TestSDKHandshakeAuthAndIdentity(t *testing.T) {
	f := newFakeNative(t, true)
	c := dialFake(t, f)
	if c.ServerInfo().SelectedProtocol != 1 {
		t.Fatal("invalid version")
	}
	if err := c.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, err := DialWithOptions(ctx, f.url(), DialOptions{APIKey: "bad", Timeout: time.Second})
	if !IsCode(err, "UNAUTHORIZED") {
		t.Fatalf("expected UNAUTHORIZED: %v", err)
	}
	_, err = DialWithOptions(ctx, f.url(), DialOptions{APIKey: "test-key", ExpectedIdentityKey: base64.RawStdEncoding.EncodeToString(make([]byte, 32))})
	if err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("expected pin mismatch: %v", err)
	}
	_, err = Dial(ctx, f.url())
	if err == nil || !strings.Contains(err.Error(), "requires API key") {
		t.Fatalf("expected auth required: %v", err)
	}
}
func TestSDKSnapshotPreservesBigInts(t *testing.T) {
	c := dialFake(t, newFakeNative(t, false))
	snapshot, err := c.GetSnapshot(context.Background(), "big")
	if err != nil {
		t.Fatal(err)
	}
	if string(snapshot.Value) != "18446744073709551615" {
		t.Fatalf("lost integer precision: %s", snapshot.Value)
	}
	var n json.Number
	if err := snapshot.Decode(&n); err != nil {
		t.Fatal(err)
	}
	if n.String() != "18446744073709551615" {
		t.Fatal(n)
	}
}
func TestSDKConcurrentMultiplexing(t *testing.T) {
	c := dialFake(t, newFakeNative(t, false))
	const count = 100
	var wg sync.WaitGroup
	errs := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var r map[string]any
			err := c.Command(context.Background(), "ECHO_DELAY", map[string]any{"delay_ms": i % 5}, &r)
			if err != nil {
				errs <- err
				return
			}
			if r["id"] == nil {
				errs <- errors.New("missing response id")
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
func TestSDKErrorAndLifecycle(t *testing.T) {
	c := dialFake(t, newFakeNative(t, false))
	ctx := context.Background()
	var out int64
	err := c.Op(ctx, "a", "FAIL", nil, &out)
	if !IsCode(err, "WRONG_TYPE") {
		t.Fatalf("unexpected error %v", err)
	}
	if err := c.CompareAndSet(ctx, "x", 0, "integer", 1, WriteOptions{}); err == nil {
		t.Fatal("zero version")
	}
	if err := c.SetWithOptions(ctx, "k", "integer", 1, WriteOptions{NX: true, XX: true}); err == nil {
		t.Fatal("conflicting flags")
	}
	if v, err := c.Exists(ctx, "a"); err != nil || !v {
		t.Fatalf("exists %v %v", v, err)
	}
	if v, err := c.TypeOf(ctx, "a"); err != nil || v != "unsigned_integer" {
		t.Fatalf("type %v %v", v, err)
	}
	if ttl, err := c.TTL(ctx, "a"); err != nil || ttl != -1 {
		t.Fatalf("ttl %v %v", ttl, err)
	}
	if ts, err := c.Types(ctx); err != nil || len(ts) != 91 {
		t.Fatalf("types len=%d err=%v", len(ts), err)
	}
	if keys, err := c.Keys(ctx, "", 20); err != nil || len(keys) != 2 {
		t.Fatalf("keys %v %v", keys, err)
	}
	if page, err := c.Scan(ctx, "", "", 50); err != nil || len(page.Keys) != 2 {
		t.Fatalf("scan %v %v", page, err)
	}
	var seen []string
	if err := c.ScanAll(ctx, "", 50, func(s string) error { seen = append(seen, s); return nil }); err != nil || !reflect.DeepEqual(seen, []string{"a", "b"}) {
		t.Fatalf("scanall %v %v", seen, err)
	}
	if _, err := c.Batch(ctx, nil, false); err == nil {
		t.Fatal("batch should reject empty")
	}
	if _, err := c.Batch(ctx, []BatchRequest{{Command: "PING"}}, true); err != nil {
		t.Fatal(err)
	}
	var opResult uint64
	if err := c.TypeList("a").PushBack(ctx, map[string]any{"value": Typed("integer", 2)}, &opResult); err != nil || opResult != 22 {
		t.Fatal(opResult, err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Ping(ctx); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("expected net.ErrClosed: %v", err)
	}
}
func TestSDKPubSubLifecycle(t *testing.T) {
	c := dialFake(t, newFakeNative(t, false))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sub, err := c.Subscribe(ctx, "topic", "events", 8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Subscribe(ctx, "topic", "events", 8); err == nil {
		t.Fatal("duplicate subscription accepted on one connection")
	}
	select {
	case event, ok := <-sub.C:
		if !ok || event.Channel != "events" || event.Payload != "event-data" {
			t.Fatalf("unexpected subscription event %#v (open=%v)", event, ok)
		}
	case <-ctx.Done():
		t.Fatal("event was not routed to subscription")
	}
	if err := sub.Close(); err != nil {
		t.Fatal(err)
	}
	if err := sub.Close(); err != nil {
		t.Fatal("second close should be safe:", err)
	}
	select {
	case _, ok := <-sub.C:
		if ok {
			t.Fatal("subscription still active after unsubscribe")
		}
	default:
		t.Fatal("subscription channel not closed")
	}
}

func TestSDKBlobRoundTrip(t *testing.T) {
	f := newFakeNative(t, false)
	c := dialFake(t, f)
	const size = 257 * 1024
	payload := strings.Repeat("abcxyz\x00", size/7)
	ctx := context.Background()
	if err := c.UploadBlobWithOptions(ctx, "obj", strings.NewReader(payload), int64(len(payload)), BlobOptions{ChunkSize: 4096, ConnectionScoped: true}); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	var uploaded []byte
	for _, v := range f.blobs {
		uploaded = v
	}
	f.mu.Unlock()
	if !reflect.DeepEqual(uploaded, []byte(payload)) {
		t.Fatal("uploaded bytes differ")
	}
	// The fake server stores uploads by upload ID. Read via that ID.
	f.mu.Lock()
	var id string
	for k := range f.blobs {
		id = k
	}
	f.mu.Unlock()
	var out strings.Builder
	total, err := c.DownloadBlob(ctx, id, &out, 8192)
	if err != nil || total != int64(len(payload)) || out.String() != payload {
		t.Fatalf("download total=%d err=%v equal=%v", total, err, out.String() == payload)
	}
	if err := c.UploadBlobWithOptions(ctx, "x", strings.NewReader("short"), 15, BlobOptions{ChunkSize: 4}); !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		t.Fatalf("short reader should fail: %v", err)
	}
	if _, err := c.BlobChunk(ctx, "id", 0, nil); err == nil {
		t.Fatal("empty chunk accepted")
	}
}
func TestSDKTypedRegistry(t *testing.T) {
	names := RegisteredTypes()
	if len(names) != 91 {
		t.Fatalf("got %d types", len(names))
	}
	ops := 0
	for _, typ := range names {
		ops += len(DocumentedOperations(typ))
	}
	if ops != 242 {
		t.Fatalf("got %d typed OP combinations", ops)
	}
	if len(DocumentedOperations("binary_string")) != 0 {
		t.Fatal("binary_string GET is a top-level command")
	}
	if got := DocumentedOperations("list"); !reflect.DeepEqual(got, []string{"PUSH_BACK", "PUSH_FRONT", "LEN", "RANGE", "POP_FRONT", "POP_BACK"}) {
		t.Fatal(got)
	}
}
func BenchmarkSDKRequestRoundTrip(b *testing.B) {
	// net.Pipe is intentionally not used: this tests request framing and socket multiplexing.
	b.StopTimer()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, e := ln.Accept()
			if e != nil {
				return
			}
			go func() {
				defer conn.Close()
				for {
					f, e := protocol.ReadFrame(conn)
					if e != nil {
						return
					}
					r, _ := json.Marshal(protocol.Response{OK: true, Result: "PONG"})
					_ = protocol.WriteFrame(conn, protocol.Frame{Kind: protocol.KindResponse, RequestID: f.RequestID, Payload: r})
				}
			}()
		}
	}()
	ctx := context.Background()
	c, err := Dial(ctx, "amaquet://"+ln.Addr().String())
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		if err := c.Ping(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
