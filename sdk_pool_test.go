package amaquet

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestSDKConnectionPool(t *testing.T) {
	srv := newFakeNative(t, false)
	ctx := context.Background()
	if _, err := DialPool(ctx, srv.url(), 0, DialOptions{}); err == nil {
		t.Fatal("accepted empty pool")
	}
	p, err := DialPool(ctx, srv.url(), 4, DialOptions{RequireIdentity: true})
	if err != nil {
		t.Fatal(err)
	}
	if p.Size() != 4 {
		t.Fatal(p.Size())
	}
	for i := 0; i < 50; i++ {
		if err := p.Command(ctx, "PING", nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	c, err := p.Borrow(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := p.Command(ctx, "PING", nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Borrow(ctx); !errors.Is(err, net.ErrClosed) {
		t.Fatal(err)
	}
}
