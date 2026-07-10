// Package engine is the gateway's client to the C++ matching engine over gRPC.
//
// In Phase 3 the gateway pulls live fills straight from the engine's StreamFills
// RPC. Phase 4 replaces this source with a Kafka consumer; everything above this
// package (the hub, the WebSocket layer) stays unchanged because it only sees a
// channel of fills.
package engine

import (
	"context"

	"github.com/shubham/trading-platform/gateway-go/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	engine pb.MatchingEngineClient
}

// Dial connects to the engine. Plaintext locally; TLS is a tracked Phase 6 gap.
func Dial(target string) (*Client, error) {
	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, engine: pb.NewMatchingEngineClient(conn)}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

// GetBookSnapshot returns the top `depth` levels (0 = full book).
func (c *Client) GetBookSnapshot(ctx context.Context, symbol string, depth uint32) (*pb.BookSnapshot, error) {
	return c.engine.GetBookSnapshot(ctx, &pb.GetBookSnapshotRequest{
		Symbol: symbol,
		Depth:  depth,
	})
}

// StreamFills opens the server-streaming RPC for a symbol and returns a channel
// of fills. The channel is closed when ctx is cancelled or the stream ends; the
// caller learns of a terminal error via the returned <-chan error (buffered,
// one value).
func (c *Client) StreamFills(ctx context.Context, symbol string) (<-chan *pb.Fill, <-chan error, error) {
	stream, err := c.engine.StreamFills(ctx, &pb.StreamFillsRequest{Symbol: symbol})
	if err != nil {
		return nil, nil, err
	}

	fills := make(chan *pb.Fill)
	errc := make(chan error, 1)

	go func() {
		defer close(fills)
		for {
			fill, err := stream.Recv()
			if err != nil {
				errc <- err // io.EOF on a clean close
				return
			}
			select {
			case fills <- fill:
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			}
		}
	}()

	return fills, errc, nil
}
