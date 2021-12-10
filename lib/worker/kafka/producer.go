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
	"time"
)

type Producer struct {
	writer *kafka.Writer
	ctx    context.Context
	debug  bool
	topic  string
}

func NewProducer(ctx context.Context, bootstrapUrl string, topic string, debug bool) (*Producer, error) {
	result := &Producer{ctx: ctx, debug: debug, topic: topic}
	broker, err := GetBroker(bootstrapUrl)
	if err != nil {
		log.Println("ERROR: unable to get broker list", err)
		return nil, err
	}
	err = InitTopic(bootstrapUrl, topic)
	if err != nil {
		log.Println("ERROR: unable to create topic", err)
		return nil, err
	}
	var logger *log.Logger = nil

	if debug {
		logger = log.New(os.Stdout, "KAFKA", 0)
	}

	result.writer = &kafka.Writer{
		Addr:        kafka.TCP(broker...),
		Topic:       topic,
		MaxAttempts: 10,
		Logger:      logger,
		BatchSize:   1,
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
