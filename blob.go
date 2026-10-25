package amaquet

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"
)

// MaxBlobSize is the largest blob size accepted by the SDK.
const MaxBlobSize int64 = 1 << 30

// MaxBlobReadSize is the largest chunk or range size accepted by the SDK.
const MaxBlobReadSize = 32 << 20

// BlobOptions: Provides the Blob Options operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Implements staged binary transfer commands with upload IDs, chunk offsets, range validation, and commit or abort lifecycle.
// Binary transfers use staged IDs and bounded chunks; abort incomplete uploads and verify range limits.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type BlobOptions struct {
	// UploadID identifies an existing staged upload; leave it empty to generate one.
	UploadID string
	// ChunkSize is the upload chunk size used by UploadBlobWithOptions.
	ChunkSize int
	// TTL is the requested time-to-live for the resulting blob.
	TTL time.Duration
	// NX requests creation only when the key does not already exist.
	NX bool
	// XX requests writing only when the key already exists.
	XX bool
	// ConnectionScoped limits the staged upload to the connection that created it.
	ConnectionScoped bool
	// ExpiresAt is the absolute expiration time for the resulting blob.
	ExpiresAt time.Time
}

// BlobUpload: Provides the Blob Upload operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Implements staged binary transfer commands with upload IDs, chunk offsets, range validation, and commit or abort lifecycle.
// Binary transfers use staged IDs and bounded chunks; abort incomplete uploads and verify range limits.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type BlobUpload struct {
	// UploadID identifies the staged upload.
	UploadID string `json:"upload_id"`
	// TotalSize is the expected total number of bytes.
	TotalSize int64 `json:"total_size"`
	// ExpiresInMS is the remaining lifetime of the staged upload in milliseconds.
	ExpiresInMS int64 `json:"expires_in_ms"`
	// ConnectionScoped reports whether the upload is bound to its creating connection.
	ConnectionScoped bool `json:"connection_scoped"`
}

// blobID: Provides the blob ID operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Binary transfers use staged IDs and bounded chunks; abort incomplete uploads and verify range limits.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func blobID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(b[:]), nil
}

// BeginBlob: Provides the Begin Blob operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Binary transfers use staged IDs and bounded chunks; abort incomplete uploads and verify range limits.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) BeginBlob(ctx context.Context, key string, size int64, opts BlobOptions) (BlobUpload, error) {
	if size < 0 || size > MaxBlobSize {
		return BlobUpload{}, fmt.Errorf("blob size must be between 0 and %d", MaxBlobSize)
	}

	id := opts.UploadID
	if id == "" {
		var err error
		id, err = blobID()

		if err != nil {
			return BlobUpload{}, err
		}
	}

	a := map[string]any{
		"key":               key,
		"upload_id":         id,
		"total_size":        size,
		"connection_scoped": opts.ConnectionScoped,
	}

	if err := (WriteOptions{
		TTL:       opts.TTL,
		NX:        opts.NX,
		XX:        opts.XX,
		ExpiresAt: opts.ExpiresAt,
	}).apply(a); err != nil {
		return BlobUpload{}, err
	}

	var result BlobUpload
	err := c.Command(ctx, "BLOB_BEGIN", a, &result)

	return result, err
}

// BlobChunk: Provides the Blob Chunk operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Binary transfers use staged IDs and bounded chunks; abort incomplete uploads and verify range limits.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) BlobChunk(ctx context.Context, uploadID string, offset int64, data []byte) (int64, error) {
	if len(data) == 0 || len(data) > MaxBlobReadSize || offset < 0 {
		return 0, errors.New("invalid blob chunk: expected nonempty data of at most 32 MiB and nonnegative offset")
	}

	var result struct {
		Received  int64 `json:"received"`
		TotalSize int64 `json:"total_size"`
	}

	err := c.Command(ctx, "BLOB_CHUNK", map[string]any{
		"upload_id": uploadID,
		"offset":    offset,
		"data":      base64.StdEncoding.EncodeToString(data),
	}, &result)
	return result.Received, err
}

// CommitBlob: Provides the Commit Blob operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Binary transfers use staged IDs and bounded chunks; abort incomplete uploads and verify range limits.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) CommitBlob(ctx context.Context, id string) error {
	return c.Command(ctx, "BLOB_COMMIT", map[string]any{"upload_id": id}, nil)
}

// AbortBlob: Provides the Abort Blob operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Binary transfers use staged IDs and bounded chunks; abort incomplete uploads and verify range limits.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) AbortBlob(ctx context.Context, id string) error {
	return c.Command(ctx, "BLOB_ABORT", map[string]any{"upload_id": id}, nil)
}

// UploadBlobWithOptions reads exactly size bytes and aborts a failed transfer.
// Reusing a completed upload ID is not safe; leave UploadID blank for a random 128-bit ID.
// UploadBlobWithOptions: Provides the Upload Blob With Options operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Binary transfers use staged IDs and bounded chunks; abort incomplete uploads and verify range limits.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) UploadBlobWithOptions(ctx context.Context, key string, r io.Reader, size int64, opts BlobOptions) error {
	if r == nil {
		return errors.New("reader is nil")
	}

	if size < 0 || size > MaxBlobSize {
		return fmt.Errorf("blob size must be between 0 and %d", MaxBlobSize)
	}

	chunk := opts.ChunkSize
	if chunk <= 0 {
		chunk = 1 << 20
	}

	if chunk > MaxBlobReadSize {
		return errors.New("chunk size exceeds 32 MiB")
	}

	if size > 0 && (size+int64(chunk)-1)/int64(chunk) > 65536 {
		return errors.New("upload would exceed the server maximum of 65536 non-overlapping chunks")
	}

	// Preflight avoids starting uploads with invalid write options.
	if err := (WriteOptions{
		TTL:       opts.TTL,
		NX:        opts.NX,
		XX:        opts.XX,
		ExpiresAt: opts.ExpiresAt,
	}).apply(map[string]any{}); err != nil {
		return err
	}

	begin, err := c.BeginBlob(ctx, key, size, opts)
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			cleanCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_ = c.AbortBlob(cleanCtx, begin.UploadID)
		}
	}()

	b := make([]byte, chunk)
	for off := int64(0); off < size; {
		n := int64(len(b))
		if size-off < n {
			n = size - off
		}

		_, err = io.ReadFull(r, b[:int(n)])
		if err != nil {
			return err
		}

		if _, err = c.BlobChunk(ctx, begin.UploadID, off, b[:int(n)]); err != nil {
			return err
		}

		off += n
	}

	if err = c.CommitBlob(ctx, begin.UploadID); err != nil {
		return err
	}

	committed = true
	return nil
}

// DownloadBlob reads the object through bounded chunks and writes all bytes to dst.
// A concurrent mutation can cause a size/version mismatch; callers that need an
// immutable read should coordinate writes at application level.
// DownloadBlob: Provides the Download Blob operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Binary transfers use staged IDs and bounded chunks; abort incomplete uploads and verify range limits.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) DownloadBlob(ctx context.Context, key string, dst io.Writer, chunkSize int) (int64, error) {
	if dst == nil {
		return 0, errors.New("writer is nil")
	}

	if chunkSize <= 0 {
		chunkSize = 1 << 20
	}

	if chunkSize > MaxBlobReadSize {
		return 0, errors.New("chunk size exceeds 32 MiB")
	}

	total := int64(-1)

	var version uint64
	for pos := int64(0); total < 0 || pos < total; {
		if err := ctx.Err(); err != nil {
			return pos, err
		}

		rng, err := c.BlobReadRange(ctx, key, pos, chunkSize)
		if err != nil {
			return pos, err
		}

		if total < 0 {
			total = rng.Size
			version = rng.Version
		} else if rng.Size != total || rng.Version != version {
			return pos, errors.New("blob changed during download")
		}

		if rng.Key != key || rng.Offset != pos {
			return pos, errors.New("server returned inconsistent blob range metadata")
		}

		if int64(len(rng.Data)) > total-pos {
			return pos, errors.New("server returned bytes past end of blob")
		}

		if len(rng.Data) == 0 && pos < total {
			return pos, io.ErrUnexpectedEOF
		}

		n, err := io.Copy(dst, bytes.NewReader(rng.Data))
		pos += n

		if err != nil {
			return pos, err
		}
	}

	return total, nil
}

// BlobRange carries complete BLOB_READ metadata including the key version.
// BlobRange: Provides the Blob Range operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Implements staged binary transfer commands with upload IDs, chunk offsets, range validation, and commit or abort lifecycle.
// Binary transfers use staged IDs and bounded chunks; abort incomplete uploads and verify range limits.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type BlobRange struct {
	// Key is the key from which the range was read.
	Key string
	// Version is the key version associated with the range.
	Version uint64
	// Offset is the starting byte offset of Data.
	Offset int64
	// Size is the number of bytes returned in Data.
	Size int64
	// Data contains the requested blob bytes.
	Data []byte
}

// BlobReadRange reads and checks a server response without losing its version metadata.
// BlobReadRange: Provides the Blob Read Range operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Binary transfers use staged IDs and bounded chunks; abort incomplete uploads and verify range limits.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (c *Client) BlobReadRange(ctx context.Context, key string, offset int64, length int) (BlobRange, error) {
	if offset < 0 || length < 0 || length > MaxBlobReadSize {
		return BlobRange{}, errors.New("invalid blob range")
	}

	var result struct {
		Key     string `json:"key"`
		Version uint64 `json:"version"`
		Offset  int64  `json:"offset"`
		Size    int64  `json:"size"`
		Data    string `json:"data"`
	}

	if err := c.Command(ctx, "BLOB_READ", map[string]any{
		"key":    key,
		"offset": offset,
		"length": length,
	}, &result); err != nil {
		return BlobRange{}, err
	}

	data, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		return BlobRange{}, err
	}

	return BlobRange{
		Key:     result.Key,
		Version: result.Version,
		Offset:  result.Offset,
		Size:    result.Size,
		Data:    data,
	}, nil
}
