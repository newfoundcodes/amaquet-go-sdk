// semaphore: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/semaphore
// Optional: go run ./examples/topics/semaphore -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:semaphore:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeSemaphore(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{"limit":2}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as semaphore\n", key)
	if *operation != "" {
		switch *operation {
		case "TRY_ACQUIRE":
			var result json.RawMessage
			common.Must(handle.TryAcquire(ctx, common.Args(`{}`), &result))
			common.Print("TRY_ACQUIRE", result)

		case "STATS":
			var result json.RawMessage
			common.Must(handle.Stats(ctx, common.Args(`{}`), &result))
			common.Print("STATS", result)

		case "RELEASE":
			var result json.RawMessage
			common.Must(handle.Release(ctx, common.Args(`{}`), &result))
			common.Print("RELEASE", result)

		case "ACQUIRE":
			var result json.RawMessage
			common.Must(handle.Acquire(ctx, common.Args(`{}`), &result))
			common.Print("ACQUIRE", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
