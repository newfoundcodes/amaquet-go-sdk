// null: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/null
// Optional: go run ./examples/topics/null -op <DOCUMENTED_OPERATION>
package main

import (
	"context"
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

	key := fmt.Sprintf("sdk:topic:null:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeNull(key)
	common.Must(handle.Operation.Set(ctx, common.Args(`{"value":null}`)["value"], amaquet.WriteOptions{}))

	fmt.Printf("Created %s as null\n", key)
	if *operation != "" {
		switch *operation {
		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
