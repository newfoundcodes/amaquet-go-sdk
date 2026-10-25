// server_information: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/server_information
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("server_information", func(ctx context.Context, c *amaquet.Client, key string) error {
		info, err := c.Info(ctx)
		if err == nil {
			common.Print("server info", info)
		}

		return err
	})
}
