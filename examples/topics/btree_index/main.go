// btree_index: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/btree_index
// Optional: go run ./examples/topics/btree_index -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:btree_index:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeBtreeIndex(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{"degree":8}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as btree_index\n", key)
	if *operation != "" {
		switch *operation {
		case "INSERT":
			var result json.RawMessage
			common.Must(handle.Insert(ctx, common.Args(`{"value":"b","id":"2"}`), &result))
			common.Print("INSERT", result)

		case "SEARCH":
			var result json.RawMessage
			common.Must(handle.Search(ctx, common.Args(`{"value":"a"}`), &result))
			common.Print("SEARCH", result)

		case "DELETE":
			var result json.RawMessage
			common.Must(handle.Delete(ctx, common.Args(`{"value":"b","id":"2"}`), &result))
			common.Print("DELETE", result)

		case "RANGE":
			var result json.RawMessage
			common.Must(handle.Range(ctx, common.Args(`{"value":"a","max":"z","limit":10}`), &result))
			common.Print("RANGE", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
