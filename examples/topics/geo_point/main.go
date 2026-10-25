// geo_point: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/geo_point
// Optional: go run ./examples/topics/geo_point -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:geo_point:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeGeoPoint(key)
	common.Must(handle.Operation.Set(ctx, common.Args(`{"value":{"Lat":14.5995,"Lon":120.9842}}`)["value"], amaquet.WriteOptions{}))

	fmt.Printf("Created %s as geo_point\n", key)
	if *operation != "" {
		switch *operation {
		case "DISTANCE":
			var result json.RawMessage
			common.Must(handle.Distance(ctx, common.Args(`{"point":{"Lat":14.5995,"Lon":120.9842}}`), &result))
			common.Print("DISTANCE", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
