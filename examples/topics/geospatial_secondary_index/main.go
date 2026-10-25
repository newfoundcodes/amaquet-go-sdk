// geospatial_secondary_index: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/geospatial_secondary_index
// Optional: go run ./examples/topics/geospatial_secondary_index -op <DOCUMENTED_OPERATION>
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

	key := fmt.Sprintf("sdk:topic:geospatial_secondary_index:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeGeospatialSecondaryIndex(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as geospatial_secondary_index\n", key)
	if *operation != "" {
		switch *operation {
		case "ADD":
			var result json.RawMessage
			common.Must(handle.Add(ctx, common.Args(`{"id":"manila","point":{"Lat":14.5995,"Lon":120.9842},"metadata":{"country":"PH"}}`), &result))
			common.Print("ADD", result)

		case "RADIUS":
			var result json.RawMessage
			common.Must(handle.Radius(ctx, common.Args(`{"center":{"Lat":14.5995,"Lon":120.9842},"radius_meters":100,"limit":5}`), &result))
			common.Print("RADIUS", result)

		case "REMOVE":
			var result json.RawMessage
			common.Must(handle.Remove(ctx, common.Args(`{"id":"manila"}`), &result))
			common.Print("REMOVE", result)

		case "BOX":
			var result json.RawMessage
			common.Must(handle.Box(ctx, common.Args(`{"box":{"MinLat":14.0,"MinLon":120.0,"MaxLat":15.0,"MaxLon":122.0}}`), &result))
			common.Print("BOX", result)

		case "POLYGON":
			var result json.RawMessage
			common.Must(handle.Polygon(ctx, common.Args(`{"points":[{"Lat":14.5,"Lon":120.9},{"Lat":14.7,"Lon":120.9},{"Lat":14.6,"Lon":121.1}]}`), &result))
			common.Print("POLYGON", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
