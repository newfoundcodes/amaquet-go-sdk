package common

import (
	"context"
	"fmt"
	"time"

	amaquet "github.com/newfoundcodes/amaquet-go-sdk"
)

// RunExample opens an authenticated connection, uses a unique key, and removes
// the key after the callback. No failed write is retried automatically.
// It is for examples against a disposable Amaquet server, not production data.
func RunExample(name string, fn func(context.Context, *amaquet.Client, string) error) {
	ctx, cancel := Context()
	defer cancel()

	client, err := Open(ctx)
	Must(err)
	defer client.Close()

	key := fmt.Sprintf("sdk:workflow:%s:%d", name, time.Now().UnixNano())
	err = fn(ctx, client, key)

	cleanupCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
	_, cleanupErr := client.Delete(cleanupCtx, key)
	done()

	if err != nil {
		Must(err)
	}

	Must(cleanupErr)
}
