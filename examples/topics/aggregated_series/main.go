// aggregated_series: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/aggregated_series
// Optional: go run ./examples/topics/aggregated_series -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:aggregated_series:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeAggregatedSeries(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{"bucket_ms":1000,"aggregation":"avg"}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as aggregated_series\n", key)
	if *operation != "" {
		switch *operation {
		case "AGGREGATE":
			var result json.RawMessage
			common.Must(handle.Aggregate(ctx, common.Args(`{"source_key":"example:timeseries:source","from":0,"to":3000,"bucket_ms":1000,"aggregation":"avg"}`), &result))
			common.Print("AGGREGATE", result)

		case "RANGE":
			var result json.RawMessage
			common.Must(handle.Range(ctx, common.Args(`{"source_key":"example:timeseries:source","from":0,"to":3000,"bucket_ms":1000,"aggregation":"avg"}`), &result))
			common.Print("RANGE", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
