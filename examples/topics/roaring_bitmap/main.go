// roaring_bitmap: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/roaring_bitmap
// Optional: go run ./examples/topics/roaring_bitmap -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:roaring_bitmap:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeRoaringBitmap(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as roaring_bitmap\n", key)
	if *operation != "" {
		switch *operation {
		case "ADD":
			var result json.RawMessage
			common.Must(handle.Add(ctx, common.Args(`{"value":42}`), &result))
			common.Print("ADD", result)

		case "CONTAINS":
			var result json.RawMessage
			common.Must(handle.Contains(ctx, common.Args(`{"value":42}`), &result))
			common.Print("CONTAINS", result)

		case "CARDINALITY":
			var result json.RawMessage
			common.Must(handle.Cardinality(ctx, common.Args(`{}`), &result))
			common.Print("CARDINALITY", result)

		case "REMOVE":
			var result json.RawMessage
			common.Must(handle.Remove(ctx, common.Args(`{"value":42}`), &result))
			common.Print("REMOVE", result)

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
