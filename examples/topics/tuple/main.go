// tuple: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/tuple
// Optional: go run ./examples/topics/tuple -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:tuple:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeTuple(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{"values":[{"type":"utf8_string","value":"alpha"},{"type":"integer","value":7}]}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as tuple\n", key)
	if *operation != "" {
		switch *operation {
		case "LEN":
			var result json.RawMessage
			common.Must(handle.Len(ctx, common.Args(`{}`), &result))
			common.Print("LEN", result)

		case "GET":
			var result json.RawMessage
			common.Must(handle.Get(ctx, common.Args(`{"index":0}`), &result))
			common.Print("GET", result)

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
