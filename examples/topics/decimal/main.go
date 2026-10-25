// decimal: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/decimal
// Optional: go run ./examples/topics/decimal -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:decimal:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeDecimal(key)
	common.Must(handle.Operation.Set(ctx, common.Args(`{"value":"12345.6789"}`)["value"], amaquet.WriteOptions{}))

	fmt.Printf("Created %s as decimal\n", key)
	if *operation != "" {
		switch *operation {
		case "ADD":
			var result json.RawMessage
			common.Must(handle.Add(ctx, common.Args(`{"value":"0.3211"}`), &result))
			common.Print("ADD", result)

		case "CMP":
			var result json.RawMessage
			common.Must(handle.Cmp(ctx, common.Args(`{"value":"12346.0000"}`), &result))
			common.Print("CMP", result)

		case "SUB":
			var result json.RawMessage
			common.Must(handle.Sub(ctx, common.Args(`{"value":"0.3211"}`), &result))
			common.Print("SUB", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
