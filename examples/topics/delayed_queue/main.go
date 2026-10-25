// delayed_queue: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/delayed_queue
// Optional: go run ./examples/topics/delayed_queue -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:delayed_queue:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeDelayedQueue(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as delayed_queue\n", key)
	if *operation != "" {
		switch *operation {
		case "ENQUEUE":
			var result json.RawMessage
			common.Must(handle.Enqueue(ctx, common.Args(`{"id":"job1","value":{"type":"utf8_string","value":"payload"},"available_at_ms":0}`), &result))
			common.Print("ENQUEUE", result)

		case "POP_READY":
			var result json.RawMessage
			common.Must(handle.PopReady(ctx, common.Args(`{}`), &result))
			common.Print("POP_READY", result)

		case "LEN":
			var result json.RawMessage
			common.Must(handle.Len(ctx, common.Args(`{}`), &result))
			common.Print("LEN", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
