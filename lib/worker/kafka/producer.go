/*
 * Copyright 2020 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kafka

import (
	"context"
	"github.com/segmentio/kafka-go"
	"log"
	"os"
	"strings"
	"time"
)

type Producer struct {
	writer *kafka.Writer
	ctx    context.Context
	debug  bool
	topic  string
}

func NewProducer(ctx context.Context, bootstrapUrl string, topic string, debug bool) (*Producer, error) {
	return NewProducerWithBalancer(ctx, bootstrapUrl, topic, debug, &kafka.Hash{})
}

func NewProducerWithKeySeparationBalancer(ctx context.Context, bootstrapUrl string, topic string, debug bool) (*Producer, error) {
	return NewProducerWithBalancer(ctx, bootstrapUrl, topic, debug, &KeySeparationBalancer{SubBalancer: &kafka.Hash{}, Seperator: "/"})
}

func NewProducerWithBalancer(ctx context.Context, broker string, topic string, debug bool, balancer kafka.Balancer) (*Producer, error) {
	result := &Producer{ctx: ctx, debug: debug, topic: topic}

	err := InitTopic(broker, topic)
	if err != nil {
		log.Println("ERROR: unable to create topic", err)
		return nil, err
	}

	var logger kafka.Logger
	if debug {
		logger = log.New(os.Stdout, "KAFKA", 0)
	}

	result.writer = &kafka.Writer{
		Addr:        kafka.TCP(broker),
		Topic:       topic,
		MaxAttempts: 10,
		Logger:      logger,
		ErrorLogger: log.New(os.Stderr, "KAFKA", 0),
		Async:       false,
		BatchSize:   1,
		Balancer:    balancer,
	}
	go func() {
		<-ctx.Done()
		result.writer.Close()
	}()
	return result, nil
}

func (this *Producer) Produce(key string, message []byte) error {
	if this.debug {
		log.Println("produce:", this.topic, key, string(message))
	}
	return this.writer.WriteMessages(this.ctx, kafka.Message{
		Key:   []byte(key),
		Value: message,
		Time:  time.Now(),
	})
}

type KeySeparationBalancer struct {
	SubBalancer kafka.Balancer
	Seperator   string
}

func (this *KeySeparationBalancer) Balance(msg kafka.Message, partitions ...int) (partition int) {
	key := string(msg.Key)
	if this.Seperator != "" {
		keyParts := strings.Split(key, this.Seperator)
		key = keyParts[0]
	}
	msg.Key = []byte(key)
	return this.SubBalancer.Balance(msg, partitions...)
}
