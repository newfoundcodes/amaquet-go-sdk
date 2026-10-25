// Example of the Amaquet geo_spatial_index type, derived from its documented wire examples.
// Run: go run ./examples/topics/geo_spatial_index
package main

import (
	"encoding/json"

	amaquet "github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	client, err := common.Open(ctx)
	common.Must(err)
	defer client.Close()

	key := "sdk:topic:geo_spatial_index"
	handle := client.TypeGeoSpatialIndex(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.
	common.Must(handle.Create(ctx, common.Args(`{}`), amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Add(ctx, common.Args(`{"id": "manila", "point": {"Lat": 14.5995, "Lon": 120.9842}, "metadata": {"country": "PH"}}`), &result))
	common.Print("geo_spatial_index ADD", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
