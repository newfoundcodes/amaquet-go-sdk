package amaquet

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

// Pool holds independent connections to ONE Amaquet server. Amaquet is single-node;
// pointing a pool at unrelated servers would NOT provide distributed consistency.
// Pool does not retry a failed command because a mutation may already have committed.
// Pool: Provides the Pool operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Reuses native connections with bounded concurrency and explicit close or replacement behavior.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type Pool struct {
	uri     string
	options DialOptions
	mu      sync.Mutex
	clients []*Client
	next    atomic.Uint64
	closed  bool
}

// DialPool creates n authenticated TCP/TLS connections. Partial construction is rolled back.
// DialPool: Provides the Dial Pool operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `uri`: Native Amaquet URI. Use amaquets:// for TLS.
// Parameter `options`: Call settings or documented construction flags, as accepted by this method.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func DialPool(ctx context.Context, uri string, n int, options DialOptions) (*Pool, error) {
	if n < 1 || n > 256 {
		return nil, errors.New("pool size must be between 1 and 256")
	}

	p := &Pool{uri: uri, options: options, clients: make([]*Client, n)}
	for i := range p.clients {
		c, err := DialWithOptions(ctx, uri, options)
		if err != nil {
			_ = p.Close()
			return nil, fmt.Errorf("pool connection %d: %w", i, err)
		}

		p.clients[i] = c
	}

	return p, nil
}

// Size: Provides the Size operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (p *Pool) Size() int {
	if p == nil {
		return 0
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.clients)
}

// Borrow rotates across connections and reopens a closed connection on demand.
// Borrow does not reserve exclusive use; Clients are safe for concurrent callers.
// Borrow: Provides the Borrow operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (p *Pool) Borrow(ctx context.Context) (*Client, error) {
	if p == nil {
		return nil, net.ErrClosed
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed || len(p.clients) == 0 {
		return nil, net.ErrClosed
	}

	i := int((p.next.Add(1) - 1) % uint64(len(p.clients)))
	c := p.clients[i]

	if c != nil {
		select {
		case <-c.closed:
		default:
			return c, nil
		}
	}

	fresh, err := DialWithOptions(ctx, p.uri, p.options)
	if err != nil {
		return nil, err
	}

	p.clients[i] = fresh
	return fresh, nil
}

// Command sends the command exactly once. Callers decide if a retry is safe.
// Command: Send a named native command with its JSON payload; this is the extension point for supported server commands.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `args`: Exact JSON argument properties supported by the server; names are not translated.
// Parameter `out`: Destination to decode the server response; preserve numeric precision.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (p *Pool) Command(ctx context.Context, command string, args map[string]any, out any) error {
	c, err := p.Borrow(ctx)
	if err != nil {
		return err
	}

	return c.Command(ctx, command, args, out)
}

// Close: Release transport resources; outstanding requests or subscriptions may stop receiving replies after closure.
// Use TLS and identity pins for remote endpoints; close resources and bound pending requests in long-running applications.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (p *Pool) Close() error {
	if p == nil {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	var err error
	for _, c := range p.clients {
		if c != nil {
			err = errors.Join(err, c.Close())
		}
	}

	return err
}
