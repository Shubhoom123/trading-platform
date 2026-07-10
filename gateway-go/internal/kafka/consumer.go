// Package kafka consumes the engine's fill stream and feeds the fan-out hub.
//
// This is the Phase 4 replacement for the Phase 3 per-symbol gRPC pump: a
// single consumer-group reader drains the shared "fills" topic and broadcasts
// each fill to the hub keyed by symbol. Everything above the hub (the WebSocket
// layer) is unchanged — it only ever saw a stream of per-symbol messages.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/shubham/trading-platform/gateway-go/internal/hub"
	"github.com/shubham/trading-platform/gateway-go/internal/metrics"
	"github.com/shubham/trading-platform/gateway-go/internal/pb"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// fillDTO is the JSON shape delivered to browsers — the wire contract, kept
// decoupled from the protobuf field names.
type fillDTO struct {
	Type         string `json:"type"`
	Symbol       string `json:"symbol"`
	PriceTicks   int64  `json:"priceTicks"`
	Quantity     uint64 `json:"quantity"`
	MakerOrderID uint64 `json:"makerOrderId"`
	TakerOrderID uint64 `json:"takerOrderId"`
	Sequence     uint64 `json:"sequence"`
}

type FillConsumer struct {
	reader  *kafkago.Reader
	hub     *hub.Hub
	metrics *metrics.Metrics
	log     *slog.Logger
}

func NewFillConsumer(brokers []string, topic, groupID string, h *hub.Hub, m *metrics.Metrics, log *slog.Logger) *FillConsumer {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID, // consumer group: offsets auto-committed by ReadMessage
		MinBytes: 1,
		MaxBytes: 10 << 20, // 10 MiB
	})
	return &FillConsumer{reader: reader, hub: h, metrics: m, log: log}
}

// Run consumes until ctx is cancelled, then closes the reader.
func (c *FillConsumer) Run(ctx context.Context) {
	defer func() {
		if err := c.reader.Close(); err != nil {
			c.log.Warn("kafka reader close", "err", err)
		}
	}()

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return
			}
			c.log.Warn("kafka read", "err", err)
			continue
		}

		var fill pb.Fill
		if err := proto.Unmarshal(msg.Value, &fill); err != nil {
			c.log.Warn("skipping malformed fill payload", "err", err)
			continue
		}

		data, err := json.Marshal(fillDTO{
			Type:         "fill",
			Symbol:       fill.GetSymbol(),
			PriceTicks:   fill.GetPriceTicks(),
			Quantity:     fill.GetQuantity(),
			MakerOrderID: fill.GetMakerOrderId(),
			TakerOrderID: fill.GetTakerOrderId(),
			Sequence:     fill.GetSequence(),
		})
		if err != nil {
			continue
		}
		c.hub.Broadcast(fill.GetSymbol(), data)
		c.metrics.FillsBroadcast.Inc()
	}
}
