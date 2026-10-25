// vector_metadata: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/vector_metadata
// Optional: go run ./examples/topics/vector_metadata -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:vector_metadata:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeVectorMetadata(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as vector_metadata\n", key)
	if *operation != "" {
		switch *operation {
		case "SET":
			var result json.RawMessage
			common.Must(handle.Set(ctx, common.Args(`{"field":"lang","value":"en"}`), &result))
			common.Print("SET", result)

		case "GET":
			var result json.RawMessage
			common.Must(handle.Get(ctx, common.Args(`{}`), &result))
			common.Print("GET", result)

		case "DELETE":
			var result json.RawMessage
			common.Must(handle.Delete(ctx, common.Args(`{"field":"lang"}`), &result))
			common.Print("DELETE", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
