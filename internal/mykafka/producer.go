package mykafka

import (
	"context"
	"encoding/json"

	"fmt"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const (
	flushTimeout = 5000
)

type Producer struct {
	producer *kafka.Producer
}

func NewProducer(address []string) (*Producer, error) {
	config := &kafka.ConfigMap{
		"bootstrap.servers": strings.Join(address, ","),
	}
	p, err := kafka.NewProducer(config)
	if err != nil {
		return nil, err
	}

	return &Producer{producer: p}, nil

}

func (p *Producer) PublishEvent(ctx context.Context, topic, key string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("kafka: json.Marshal failed: %w", err)
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          data,
		Key:            []byte(key),
	}

	deliveryChan := make(chan kafka.Event, 1)
	defer close(deliveryChan)

	if err := p.producer.Produce(msg, deliveryChan); err != nil {
		return fmt.Errorf("kafka: Produce failed: %w", err)
	}

	select {
	case e := <-deliveryChan:
		if m := e.(*kafka.Message); m.TopicPartition.Error != nil {
			return fmt.Errorf("kafka: delivery failed: %w", m.TopicPartition.Error)
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("kafka: delivery timeout")
	}

	return nil
}

func (p *Producer) Close() {
	p.producer.Flush(flushTimeout)
	p.producer.Close()
}
