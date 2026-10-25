<div align="center">
  <img src="https://amaquet.newfoundcodes.com/amaquet.png" width="128" />
  <h1>Amaquet Go SDK</h1>
  <p>
    <a href="https://github.com/newfoundcodes/amaquet-go-sdk/actions"><img src="https://github.com/newfoundcodes/amaquet-go-sdk/actions/workflows/ci.yml/badge.svg" alt="Build Status"></a>
  </p>
</div>

The standalone Go SDK connects to a running Amaquet server. It implements native TCP/TLS protocol v1, the separate HTTP administration interface, all top-level database commands, and wrappers for the registered data types. It requires Go 1.23.2 or newer and only the Go standard library.

## Start using the SDK

Use the module directly from the repository or configure your Go module to use a local checkout:

```bash
# Execute within your own Go project.
go mod edit -replace github.com/newfoundcodes/amaquet-go-sdk=/absolute/path/to/amaquet-go-sdk
go get github.com/newfoundcodes/amaquet-go-sdk
```

After the module is published at the given path with matching tags, a local replacement is not necessary.

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"

    amaquet "github.com/newfoundcodes/amaquet-go-sdk"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    db, err := amaquet.DialWithOptions(ctx, "amaquets://db.example.com:13378", amaquet.DialOptions{
        APIKey: os.Getenv("AMAQUET_API_KEY"),
        RequireIdentity: true,
        ExpectedIdentityKey: os.Getenv("AMAQUET_IDENTITY_KEY"),
    })

    if err != nil {
        panic(err)
    }
    defer db.Close()

    list := db.TypeList("sdk:example:list")
    if err := list.Create(ctx, nil, amaquet.WriteOptions{NX: true}); err != nil {
        panic(err)
    }

    var length int
    if err := list.PushBack(ctx, map[string]any{
        "value": amaquet.Typed("utf8_string", "hello"),
    }, &length); err != nil {
        panic(err)
    }

    fmt.Println("length:", length)

    snapshot, err := list.KeySnapshot(ctx)
    if err != nil {
        panic(err)
    }

    fmt.Println("value:", string(snapshot.Value))
}
```

Use `amaquet://` only for trusted plaintext environments. Use `amaquets://` with the normal Go TLS certificate and hostname verification for production. Server identity pinning adds a signed challenge check; it does not replace TLS encryption.

### Exact numbers

Do not decode monetary, 64-bit, or arbitrary-precision integer values through the old `Client.Get` map without considering float rounding. Use `GetSnapshot` and `json.RawMessage` or `Decode` with `json.Number`:

```go
snapshot, err := db.GetSnapshot(ctx, "balance")
if err != nil {
    panic(err)
}

fmt.Println(string(snapshot.Value)) // exact numeric JSON token
```

### Typed operations

Each data type has a constructor such as `db.TypeVectorSet(key)`, `db.TypeHNSWIndex(key)`, `db.TypeTimeSeries(key)`, `db.TypeReliableQueue(key)`, `db.TypeBitmap(key)`, or `db.TypeJSON(key)`. Each exposes methods named after its true `OP` operations. Method signatures take `(context.Context, map[string]any, out any)`; argument names and validation remain defined by the server's documented wire contract. This avoids accidental SDK restrictions as server-side options grow.

Seven scalar type pages mention `GET` only as a top-level key read. They do not support `OP GET`; therefore the generated facade does not invent those methods. Use `handle.KeySnapshot(ctx)` instead.

### Admin API

The admin HTTP listener is separate from the native TCP port. It supports API keys, organization, members, invitations, TOTP MFA, RBAC, sessions, persistence actions, system configuration, data inspection, audit, health, and metrics.

```go
admin, err := amaquet.NewAdminClient("https://admin.example.com:13379", amaquet.AdminOptions{
    Token: os.Getenv("AMAQUET_ADMIN_TOKEN"),
})

if err != nil {
    panic(err)
}

health, err := admin.Health(ctx)
if err != nil {
    panic(err)
}

fmt.Println(string(health))

keys, err := admin.APIKeys(ctx)
if err != nil {
    panic(err)
}

fmt.Println(string(keys))
```

The client uses request contexts, response-size limits, cookie jars, JSON number preservation, and safe default redirect behavior. `HTTPError` exposes HTTP status and structured error code. Methods return raw JSON to preserve the server's full response shape. The SDK never stores an API token in an Amaquet URI, and it does not log credentials.

### Binary transfers

The server accepts blob objects of at most 1 GiB and reads at most 32 MiB per range. The uploader checks that the planned chunks do not exceed 65,536 spans. The SDK defaults to 1 MiB buffers and uses a random 128-bit upload ID when none is supplied. `DownloadBlob` checks the object version and size across chunks and returns an error if they change. When an upload fails, a short, best-effort abort runs; an abort is not guaranteed after a process crash.

```go
source, err := os.Open("archive.bin")
if err != nil {
    panic(err)
}
defer source.Close()

stat, err := source.Stat()
if err != nil {
    panic(err)
}

err = db.UploadBlobWithOptions(ctx, "archive", source, stat.Size(), amaquet.BlobOptions{
    ChunkSize: 1 << 20, ConnectionScoped: true,
})

if err != nil {
    panic(err)
}
```

### Concurrency and failure semantics

One client uses one TCP/TLS connection with multiplexed request IDs. A background reader routes responses and Pub/Sub events, while writes use a mutex and write deadlines. Applications can call commands concurrently. A `Pool` distributes future calls across independent connections to the same server and reopens closed connections when borrowed again. **It never automatically replays a failed mutation.** A connection failure does not prove whether the server committed the operation.

`BATCH` is sequential and non-atomic. `SCAN` is not a snapshot. Live Pub/Sub can drop events when bounded queues fill, and subscriptions are not automatically restored after reconnect. Large binary reads are checked for changes while downloading, but this does not create an actual database-wide transaction. The server does not provide clustering, replication, SQL, or Redis protocol compatibility.

## Examples

Set credentials once:

```bash
export AMAQUET_URI='amaquet://127.0.0.1:13378'
export AMAQUET_API_KEY='<your-native-api-key>'
export AMAQUET_ADMIN_URL='http://127.0.0.1:13379'
export AMAQUET_ADMIN_TOKEN='<your-admin-token>'
```

The Go examples use `AMAQUET_URI` for the native URI. The equivalent C# and Java examples use `AMAQUET_ENDPOINT` and `AMAQUET_URL` respectively.

Run from **inside this standalone module**:

```bash
go run ./examples/topics/list
go run ./examples/topics/atomic_counter
go run ./examples/topics/vector_set
go run ./examples/topics/hnsw_index
go run ./examples/topics/bloom_filter
go run ./examples/topics/geo_spatial_index

go run ./examples/workflows/scan
go run ./examples/workflows/blob
go run ./examples/workflows/pubsub
go run ./examples/workflows/pool
go run ./examples/workflows/batch_cas
go run ./examples/workflows/admin_health

go run ./examples/catalog -type list
go run ./examples/catalog -type list -create -op PUSH_BACK
```

The package includes **91 executable type examples**, **42 executable workflow examples**, and 91 machine-readable type scenarios containing 242 documented operation payloads. Catalog preview is read-only. A selected operation may block, fail validation, or change existing data; do not execute unfamiliar payloads on production data.

## Testing

All standalone module tests work without the Amaquet server or third-party modules:

```bash
go test ./...
go test -race ./...
go vet ./...
go test -run '^$' -fuzz '^FuzzSDKWriteOptions$' -fuzztime=10s
go test -run '^$' -fuzz '^FuzzSDKExactJSONNumbers$' -fuzztime=10s
```

The 13 fuzz targets cover typed envelope round-trips, write options, admin IDs, unsafe paths, exact JSON numbers, binary chunk/range bounds, native frame round-trips, malformed frames, endpoint parsing, command JSON, type registries, and nonce verification. Run one fuzz target at a time. The native SDK tests use ephemeral protocol mock servers to check the real wire framing, identity handshakes, response correlation, error conversion, concurrent calls, blob transfer, and HTTP control plane.

### Signed identity fingerprint pinning

`DialOptions.ExpectedIdentityFingerprint` pins the server's Ed25519 identity as `ed25519:` followed by the lowercase SHA-256 hex digest of its 32-byte public key. The client computes this fingerprint from the signed public key and rejects mismatched advertised fingerprints. This is independent of the TLS certificate. Use `RequireIdentity` when you require signed HELLO on every connection.

For external integration testing, use `AMAQUET_LIVE_URI` and `AMAQUET_CONFIRM_DISPOSABLE=YES` on a **disposable** Amaquet instance, then run `go test -run TestLiveExternalAmaquet -v`. The test creates and removes unique keys for all 91 documented types and verifies a binary round trip.

## License

Amaquet Go SDK is licensed under the MIT License. See [LICENSE](LICENSE) for the complete license text.
