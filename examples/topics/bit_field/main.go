// bit_field: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/bit_field
// Optional: go run ./examples/topics/bit_field -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:bit_field:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()
		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeBitField(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{"size_bytes":8}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as bit_field\n", key)
	if *operation != "" {
		switch *operation {
		case "SET":
			var result json.RawMessage
			common.Must(handle.Set(ctx, common.Args(`{"offset":4,"width":8,"value":171}`), &result))
			common.Print("SET", result)

		case "GET":
			var result json.RawMessage
			common.Must(handle.Get(ctx, common.Args(`{"offset":4,"width":8}`), &result))
			common.Print("GET", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
