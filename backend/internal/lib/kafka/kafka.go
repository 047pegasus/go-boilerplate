package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/047pegasus/go-boilerplate/internal/config"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
	kafkago "github.com/segmentio/kafka-go"
)

// Producer publishes messages to Kafka, tracing each publish as a Sentry span.
type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(cfg *config.KafkaConfig) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(cfg.Brokers...),
			Balancer:     &kafkago.LeastBytes{},
			RequiredAcks: kafkago.RequireAll,
			Async:        false,
		},
	}
}

// Publish sends value (with an optional key and headers) to topic. Topic is
// set per-call rather than fixed on the Producer, since one producer can
// safely publish to many topics.
func (p *Producer) Publish(ctx context.Context, topic string, key, value []byte, headers map[string]string) error {
	span := sentry.StartSpan(ctx, "mq.kafka.publish", sentry.WithDescription(topic))
	span.SetData("messaging.system", "kafka")
	span.SetData("messaging.destination", topic)
	defer span.Finish()

	kafkaHeaders := make([]kafkago.Header, 0, len(headers))
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafkago.Header{Key: k, Value: []byte(v)})
	}

	err := p.writer.WriteMessages(span.Context(), kafkago.Message{
		Topic:   topic,
		Key:     key,
		Value:   value,
		Headers: kafkaHeaders,
	})
	if err != nil {
		span.Status = sentry.SpanStatusInternalError
		span.SetData("error", err.Error())
		return fmt.Errorf("failed to publish kafka message to %q: %w", topic, err)
	}
	span.Status = sentry.SpanStatusOK
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

// Ping verifies broker connectivity, suitable for health checks. It dials
// the first configured broker rather than any specific topic.
func Ping(ctx context.Context, cfg *config.KafkaConfig) error {
	if len(cfg.Brokers) == 0 {
		return errors.New("no kafka brokers configured")
	}
	conn, err := kafkago.DialContext(ctx, "tcp", cfg.Brokers[0])
	if err != nil {
		return fmt.Errorf("failed to reach kafka broker %q: %w", cfg.Brokers[0], err)
	}
	return conn.Close()
}

// Handler processes one Kafka message. Returning an error skips the commit,
// so the consumer group redelivers the message (at-least-once delivery).
type Handler func(ctx context.Context, msg kafkago.Message) error

// Consumer reads from a single topic within a consumer group, committing
// offsets manually only after Handler succeeds.
type Consumer struct {
	reader *kafkago.Reader
	logger *zerolog.Logger
}

func NewConsumer(cfg *config.KafkaConfig, topic string, logger *zerolog.Logger) *Consumer {
	return &Consumer{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:  cfg.Brokers,
			Topic:    topic,
			GroupID:  cfg.ConsumerGroup,
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		}),
		logger: logger,
	}
}

// Consume blocks, fetching and dispatching messages to handler until ctx is
// canceled. Run this in its own goroutine, e.g. alongside job.JobService.
func (c *Consumer) Consume(ctx context.Context, handler Handler) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("failed to fetch kafka message: %w", err)
		}

		span := sentry.StartSpan(ctx, "mq.kafka.consume", sentry.WithDescription(msg.Topic))
		span.SetData("messaging.system", "kafka")
		span.SetData("messaging.destination", msg.Topic)
		span.SetData("messaging.kafka.partition", msg.Partition)

		handleErr := handler(span.Context(), msg)
		if handleErr != nil {
			span.Status = sentry.SpanStatusInternalError
			span.SetData("error", handleErr.Error())
			span.Finish()
			c.logger.Error().Err(handleErr).Str("topic", msg.Topic).Msg("kafka handler failed, message will be redelivered")
			continue
		}

		span.Status = sentry.SpanStatusOK
		span.Finish()

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error().Err(err).Msg("failed to commit kafka message offset")
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
