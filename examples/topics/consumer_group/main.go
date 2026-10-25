// consumer_group: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/consumer_group
// Optional: go run ./examples/topics/consumer_group -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:consumer_group:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeConsumerGroup(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{"name":"workers"}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as consumer_group\n", key)
	if *operation != "" {
		switch *operation {
		case "SNAPSHOT":
			var result json.RawMessage
			common.Must(handle.Snapshot(ctx, common.Args(`{}`), &result))
			common.Print("SNAPSHOT", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
