// Package common contains small helpers for the standalone SDK examples.
package common

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	amaquet "github.com/newfoundcodes/amaquet-go-sdk"
)

func Context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func Open(ctx context.Context) (*amaquet.Client, error) {
	uri := os.Getenv("AMAQUET_URI")
	if uri == "" {
		uri = "amaquet://127.0.0.1:13378"
	}

	return amaquet.DialWithOptions(ctx, uri, amaquet.DialOptions{
		APIKey:  os.Getenv("AMAQUET_API_KEY"),
		Timeout: 10 * time.Second,
	})
}

func OpenAdmin() (*amaquet.AdminClient, error) {
	uri := os.Getenv("AMAQUET_ADMIN_URL")
	if uri == "" {
		uri = "http://127.0.0.1:13379"
	}

	return amaquet.NewAdminClient(uri, amaquet.AdminOptions{
		Token: os.Getenv("AMAQUET_ADMIN_TOKEN"),
	})
}

func Must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Amaquet error:", err)
		os.Exit(1)
	}
}

func Args(raw string) map[string]any {
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()

	var args map[string]any
	Must(dec.Decode(&args))

	return args
}

func Print(label string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	Must(err)

	fmt.Println(label+":", string(b))
}
