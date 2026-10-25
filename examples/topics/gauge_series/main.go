// gauge_series: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/gauge_series
// Optional: go run ./examples/topics/gauge_series -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:gauge_series:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeGaugeSeries(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{"labels":{"metric":"cpu"}}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as gauge_series\n", key)
	if *operation != "" {
		switch *operation {
		case "SET":
			var result json.RawMessage
			common.Must(handle.Set(ctx, common.Args(`{"timestamp":1000,"value":42.5}`), &result))
			common.Print("SET", result)

		case "LAST":
			var result json.RawMessage
			common.Must(handle.Last(ctx, common.Args(`{}`), &result))
			common.Print("LAST", result)

		case "RANGE":
			var result json.RawMessage
			common.Must(handle.Range(ctx, common.Args(`{"from":0,"to":3000}`), &result))
			common.Print("RANGE", result)

		case "AGGREGATE":
			var result json.RawMessage
			common.Must(handle.Aggregate(ctx, common.Args(`{"from":0,"to":3000,"bucket_ms":1000,"aggregation":"avg"}`), &result))
			common.Print("AGGREGATE", result)

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
