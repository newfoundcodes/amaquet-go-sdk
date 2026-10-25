package amaquet

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestLiveExternalAmaquet is opt-in. It creates and removes keys on a disposable
// server. Do not enable it against production or a shared development instance.
// Set AMAQUET_LIVE_URI and optionally AMAQUET_LIVE_API_KEY to execute it.
func TestLiveExternalAmaquet(t *testing.T) {
	uri := os.Getenv("AMAQUET_LIVE_URI")
	if uri == "" {
		t.Skip("Set AMAQUET_LIVE_URI to test a dedicated disposable Amaquet server")
	}

	if os.Getenv("AMAQUET_CONFIRM_DISPOSABLE") != "YES" {
		t.Fatal("AMAQUET_CONFIRM_DISPOSABLE=YES is required to create and delete test keys")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Minute)
	defer cancel()

	client, err := DialWithOptions(ctx, uri, DialOptions{
		APIKey:  os.Getenv("AMAQUET_LIVE_API_KEY"),
		Timeout: 30 * time.Second,
	})

	if err != nil {
		t.Fatal(err)
	}

	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}

	prefix := "sdk-audit:" + hex.EncodeToString(random[:]) + ":"
	var keys []string

	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 15*time.Second)
		defer done()

		for _, key := range keys {
			_, _ = client.Delete(cleanup, key)
		}

		_ = client.Close()
	}()

	types, err := client.Types(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(types) != 91 {
		t.Fatalf("TYPES returned %d entries; expected 91", len(types))
	}

	actual := make(map[string]bool, len(types))
	for _, name := range types {
		actual[name] = true
	}

	files, err := filepath.Glob("examples/scenarios/*.json")
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 91 {
		t.Fatalf("expected 91 scenario JSON files, got %d", len(files))
	}

	sort.Strings(files)
	for _, filename := range files {
		name := strings.TrimSuffix(filepath.Base(filename), ".json")
		if !actual[name] {
			t.Errorf("TYPES omits documented type %s", name)
			continue
		}

		t.Run(name, func(t *testing.T) {
			contents, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}

			var fixture struct {
				Type         string `json:"type"`
				Construction struct {
					Command string         `json:"command"`
					Args    map[string]any `json:"args"`
				} `json:"construction"`
			}

			decoder := json.NewDecoder(bytes.NewReader(contents))
			decoder.UseNumber() // never turn exact 64-bit integers into float64

			if err := decoder.Decode(&fixture); err != nil {
				t.Fatal(err)
			}

			if fixture.Type != name || (fixture.Construction.Command != "CREATE" && fixture.Construction.Command != "SET") {
				t.Fatalf("invalid construction fixture for %s", name)
			}

			key := prefix + name
			fixture.Construction.Args["key"] = key

			keys = append(keys, key)
			opCtx, stop := context.WithTimeout(ctx, 25*time.Second)
			defer stop()

			if err := client.Command(opCtx, fixture.Construction.Command, fixture.Construction.Args, nil); err != nil {
				t.Fatal(err)
			}

			gotType, err := client.TypeOf(opCtx, key)
			if err != nil || gotType != name {
				t.Fatalf("TYPE=%s; error=%v", gotType, err)
			}

			snapshot, err := client.GetSnapshot(opCtx, key)
			if err != nil || snapshot.Type != name || snapshot.Version == 0 {
				t.Fatalf("GET snapshot: type=%s version=%d error=%v", snapshot.Type, snapshot.Version, err)
			}
		})
	}

	key := prefix + "blob"
	keys = append(keys, key)
	payload := bytes.Repeat([]byte("\x00amqt\xff"), 400_000)

	if err := client.UploadBlobWithOptions(ctx, key, bytes.NewReader(payload), int64(len(payload)), BlobOptions{
		ChunkSize:        1 << 20,
		ConnectionScoped: true,
	}); err != nil {
		t.Fatal(fmt.Errorf("blob upload: %w", err))
	}

	var output bytes.Buffer
	size, err := client.DownloadBlob(ctx, key, &output, 1<<20)

	if err != nil || size != int64(len(payload)) || !bytes.Equal(output.Bytes(), payload) {
		t.Fatalf("blob mismatch: size=%d want=%d err=%v", size, len(payload), err)
	}
}
