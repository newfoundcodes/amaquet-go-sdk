// adjacency_set: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/adjacency_set
// Optional: go run ./examples/topics/adjacency_set -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:adjacency_set:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeAdjacencySet(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as adjacency_set\n", key)
	if *operation != "" {
		switch *operation {
		case "ADD":
			var result json.RawMessage
			common.Must(handle.Add(ctx, common.Args(`{"from":"a","to":"b","weight":1.5}`), &result))
			common.Print("ADD", result)

		case "NEIGHBORS":
			var result json.RawMessage
			common.Must(handle.Neighbors(ctx, common.Args(`{"from":"a"}`), &result))
			common.Print("NEIGHBORS", result)

		case "REMOVE":
			var result json.RawMessage
			common.Must(handle.Remove(ctx, common.Args(`{"from":"a","to":"b"}`), &result))
			common.Print("REMOVE", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
