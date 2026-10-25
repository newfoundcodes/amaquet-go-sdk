// inverted_index: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/inverted_index
// Optional: go run ./examples/topics/inverted_index -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:inverted_index:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeInvertedIndex(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as inverted_index\n", key)
	if *operation != "" {
		switch *operation {
		case "INDEX":
			var result json.RawMessage
			common.Must(handle.Index(ctx, common.Args(`{"id":"d1","text":"amaquet in-memory database"}`), &result))
			common.Print("INDEX", result)

		case "SEARCH":
			var result json.RawMessage
			common.Must(handle.Search(ctx, common.Args(`{"query":"memory","limit":10}`), &result))
			common.Print("SEARCH", result)

		case "REMOVE":
			var result json.RawMessage
			common.Must(handle.Remove(ctx, common.Args(`{"id":"d1"}`), &result))
			common.Print("REMOVE", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
