// Standalone Amaquet SDK example. Run: go run ./examples/workflows/blob
package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	c, e := common.Open(ctx)
	common.Must(e)
	defer c.Close()

	key := fmt.Sprintf("sdk:blob:%d", time.Now().UnixNano())
	data := bytes.Repeat([]byte("Amaquet\x00"), 500)

	common.Must(c.UploadBlobWithOptions(ctx, key, bytes.NewReader(data), int64(len(data)), amaquet.BlobOptions{
		ChunkSize:        4096,
		ConnectionScoped: true,
	}))

	var buf bytes.Buffer
	size, e := c.DownloadBlob(ctx, key, &buf, 4096)
	common.Must(e)

	fmt.Println("verified:", size == int64(len(data)) && bytes.Equal(data, buf.Bytes()))
}
