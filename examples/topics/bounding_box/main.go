// bounding_box: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/bounding_box
// Optional: go run ./examples/topics/bounding_box -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:bounding_box:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeBoundingBox(key)
	common.Must(handle.Operation.Set(ctx, common.Args(`{"value":{"MinLat":14.0,"MinLon":120.0,"MaxLat":15.0,"MaxLon":122.0}}`)["value"], amaquet.WriteOptions{}))

	fmt.Printf("Created %s as bounding_box\n", key)
	if *operation != "" {
		switch *operation {
		case "CONTAINS":
			var result json.RawMessage
			common.Must(handle.Contains(ctx, common.Args(`{"point":{"Lat":14.5,"Lon":121.0}}`), &result))
			common.Print("CONTAINS", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
