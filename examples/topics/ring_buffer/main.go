// ring_buffer: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/ring_buffer
// Optional: go run ./examples/topics/ring_buffer -op <DOCUMENTED_OPERATION>
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	amaquet "github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	operation := flag.String("op", "", "documented type operation; runs only when requested")
	flag.Parse()

	ctx, cancel := common.Context()
	defer cancel()

	client, err := common.Open(ctx)
	common.Must(err)
	defer client.Close()

	key := fmt.Sprintf("sdk:topic:ring_buffer:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeRingBuffer(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{"capacity":4}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as ring_buffer\n", key)
	if *operation != "" {
		switch *operation {
		case "PUSH":
			var result json.RawMessage
			common.Must(handle.Push(ctx, common.Args(`{"value":{"type":"utf8_string","value":"a"}}`), &result))
			common.Print("PUSH", result)

		case "LEN":
			var result json.RawMessage
			common.Must(handle.Len(ctx, common.Args(`{}`), &result))
			common.Print("LEN", result)

		case "VALUES":
			var result json.RawMessage
			common.Must(handle.Values(ctx, common.Args(`{}`), &result))
			common.Print("VALUES", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
