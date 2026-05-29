package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/tranvux/boilerplate_golang/pkg/config"
	"github.com/tranvux/boilerplate_golang/pkg/logger"
	"go.uber.org/zap"
)

type Producer interface {
	Publish(ctx context.Context, topic string, key []byte, value []byte) error
	Close() error
}

type kafkaProducer struct {
	writer *kafka.Writer
	log    logger.Logger
}

var _ Producer = (*kafkaProducer)(nil)

func NewKafkaProducer(cfg *config.Config, log logger.Logger) (Producer, error) {
	if len(cfg.Kafka.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers list is empty")
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		RequiredAcks: kafka.RequireAll, // Highest durability setting (acks = -1/all)
	}

	log.Info("Successfully initialized Kafka Producer", zap.Strings("brokers", cfg.Kafka.Brokers))

	return &kafkaProducer{
		writer: writer,
		log:    log,
	}, nil
}

func (p *kafkaProducer) Publish(ctx context.Context, topic string, key []byte, value []byte) error {
	p.log.Debug("Publishing message to Kafka", zap.String("topic", topic), zap.String("key", string(key)))

	err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
		Time:  time.Now(),
	})

	if err != nil {
		p.log.Error("Failed to publish message to Kafka", zap.String("topic", topic), zap.Error(err))
		return fmt.Errorf("failed to write message to topic %s: %w", topic, err)
	}

	return nil
}

func (p *kafkaProducer) Close() error {
	return p.writer.Close()
}

type MessageHandler func(ctx context.Context, msg kafka.Message) error

type ConsumerGroup interface {
	Subscribe(ctx context.Context, topic string, handler MessageHandler)
	Close() error
}

type kafkaConsumerGroup struct {
	brokers []string
	groupID string
	log     logger.Logger
	readers []*kafka.Reader
}

var _ ConsumerGroup = (*kafkaConsumerGroup)(nil)

func NewKafkaConsumerGroup(cfg *config.Config, log logger.Logger) (ConsumerGroup, error) {
	if len(cfg.Kafka.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers list is empty")
	}
	if cfg.Kafka.GroupID == "" {
		return nil, fmt.Errorf("kafka group id is empty")
	}

	return &kafkaConsumerGroup{
		brokers: cfg.Kafka.Brokers,
		groupID: cfg.Kafka.GroupID,
		log:     log,
		readers: make([]*kafka.Reader, 0),
	}, nil
}

func (c *kafkaConsumerGroup) Subscribe(ctx context.Context, topic string, handler MessageHandler) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.brokers,
		GroupID:  c.groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	c.readers = append(c.readers, reader)
	c.log.Info("Subscribed to Kafka topic", zap.String("topic", topic), zap.String("group_id", c.groupID))

	go func() {
		for {
			select {
			case <-ctx.Done():
				c.log.Info("Context cancelled, stopping consumer subscription", zap.String("topic", topic))
				return
			default:
				msg, err := reader.ReadMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					c.log.Error("Failed to read message from Kafka reader", zap.String("topic", topic), zap.Error(err))
					time.Sleep(1 * time.Second) // backoff
					continue
				}

				c.log.Debug("Received message from Kafka topic",
					zap.String("topic", topic),
					zap.Int64("offset", msg.Offset),
					zap.String("key", string(msg.Key)),
				)

				if err := handler(ctx, msg); err != nil {
					c.log.Error("Error handling Kafka message", zap.String("topic", topic), zap.Error(err))
					// In production, you would forward this message to a Dead Letter Queue (DLQ)
				}
			}
		}
	}()
}

func (c *kafkaConsumerGroup) Close() error {
	var firstErr error
	for _, reader := range c.readers {
		if err := reader.Close(); err != nil {
			c.log.Error("Error closing reader", zap.Error(err))
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}
