// text_field: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/text_field
// Optional: go run ./examples/topics/text_field -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:text_field:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeTextField(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{"name":"body","value":"hello amaquet","weight":1.5}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as text_field\n", key)
	if *operation != "" {
		switch *operation {
		case "SET":
			var result json.RawMessage
			common.Must(handle.Set(ctx, common.Args(`{"name":"body","value":"updated text","weight":2}`), &result))
			common.Print("SET", result)

		case "GET":
			var result json.RawMessage
			common.Must(handle.Get(ctx, common.Args(`{}`), &result))
			common.Print("GET", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
