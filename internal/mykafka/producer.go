package mykafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	writeTimeout = 5 * time.Second
)

type Producer struct {
	writers map[string]*kafka.Writer
}

func NewProducer(brokers []string, topics []string) (*Producer, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("no topics provided")
	}
	p := &Producer{writers: make(map[string]*kafka.Writer, len(topics))}
	for _, topic := range topics {
		p.writers[topic] = kafka.NewWriter(kafka.WriterConfig{
			Brokers:  brokers,
			Topic:    topic,
			Balancer: &kafka.Hash{},
		})
	}
	return p, nil
}

func (p *Producer) PublishEvent(ctx context.Context, topic, key string, event interface{}) error {
	w, ok := p.writers[topic]
	if !ok {
		return fmt.Errorf("unknown topic %q", topic)
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json.Marshal failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()

	msg := kafka.Message{
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	}

	if err := w.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("WriteMessages failed: %w", err)
	}
	return nil
}

func (p *Producer) Close() error {
	var firstErr error
	for topic, w := range p.writers {
		if err := w.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("closing writer %q failed: %w", topic, err)
		}
	}
	return firstErr
}
