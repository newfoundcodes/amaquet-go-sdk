// t_digest: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/t_digest
// Optional: go run ./examples/topics/t_digest -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:t_digest:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeTDigest(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{"compression":100}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as t_digest\n", key)
	if *operation != "" {
		switch *operation {
		case "ADD":
			var result json.RawMessage
			common.Must(handle.Add(ctx, common.Args(`{"value":5}`), &result))
			common.Print("ADD", result)

		case "QUANTILE":
			var result json.RawMessage
			common.Must(handle.Quantile(ctx, common.Args(`{"q":0.5}`), &result))
			common.Print("QUANTILE", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
