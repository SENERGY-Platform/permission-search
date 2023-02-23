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
	"errors"
	"github.com/segmentio/kafka-go"
	"io"
	"log"
	"os"
	"time"
)

func NewConsumer(ctx context.Context, bootstrapUrl string, groupId string, topic string, listener func(delivery []byte) error, errhandler func(err error)) error {
	broker, err := GetBroker(bootstrapUrl)
	if err != nil {
		log.Println("ERROR: unable to get broker list", err)
		return err
	}

	err = InitTopic(bootstrapUrl, topic)
	if err != nil {
		log.Println("ERROR: unable to create topic", err)
		return err
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		CommitInterval: 0, //synchronous commits
		Brokers:        broker,
		GroupID:        groupId,
		Topic:          topic,
		MaxWait:        1 * time.Second,
		Logger:         log.New(io.Discard, "", 0),
		ErrorLogger:    log.New(os.Stdout, "[KAFKA-ERROR] ", log.Default().Flags()),
	})

	go func() {
		defer r.Close()
		defer log.Println("close consumer for topic ", topic)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				m, err := r.FetchMessage(ctx)
				if err == io.EOF || err == context.Canceled {
					return
				}
				if err != nil {
					log.Println("ERROR: while consuming topic ", topic, err)
					errhandler(err)
					return
				}

				err = retry(func() error {
					return listener(m.Value)
				}, func(n int64) time.Duration {
					return time.Duration(n) * time.Second
				}, 10*time.Minute)

				if err != nil {
					log.Println("ERROR: unable to handle message (no commit)", err)
					errhandler(err)
				} else {
					err = r.CommitMessages(ctx, m)
				}
			}
		}
	}()
	return nil
}

func NewConsumerWithMultipleTopics(ctx context.Context, bootstrapUrl string, groupId string, topics []string, debug bool, listener func(topic string, delivery []byte) error, errhandler func(topice string, err error)) error {
	if len(topics) == 0 {
		return nil
	}

	log.Println("init topics:", topics)
	broker, err := GetBroker(bootstrapUrl)
	if err != nil {
		log.Println("ERROR: unable to get broker list", err)
		return err
	}

	for _, topic := range topics {
		err = InitTopic(bootstrapUrl, topic)
		if err != nil {
			log.Println("ERROR: unable to create topic", err)
			return err
		}
	}
	log.Println("consume:", topics)

	r := kafka.NewReader(kafka.ReaderConfig{
		CommitInterval: 0, //synchronous commits
		Brokers:        broker,
		GroupID:        groupId,
		GroupTopics:    topics,
		Logger:         log.New(io.Discard, "", 0),
		ErrorLogger:    log.New(os.Stdout, "[KAFKA-ERROR] ", log.Default().Flags()),
	})

	go func() {
		defer r.Close()
		defer log.Println("close consumer for topics ", topics)
		for {
			select {
			case <-ctx.Done():
				log.Println("receive ctx done for consumer of", topics)
				return
			default:
				m, err := r.FetchMessage(ctx)
				if err == io.EOF || err == context.Canceled {
					log.Println("ERROR: on fetch:", err)
					return
				}
				if debug {
					log.Println("DEBUG: receive:", m.Topic, string(m.Value))
				}
				topic := m.Topic
				if err != nil {
					log.Println("ERROR: while consuming topic ", topic, err)
					errhandler(topic, err)
					return
				}

				err = retry(func() error {
					return listener(topic, m.Value)
				}, func(n int64) time.Duration {
					return time.Duration(n) * time.Second
				}, 10*time.Minute)

				if err != nil {
					log.Println("ERROR: unable to handle message (no commit)", err)
					errhandler(topic, err)
				} else {
					err = r.CommitMessages(ctx, m)
					if err != nil {
						log.Println("ERROR: on commit:", err)
					} else {
						if debug {
							log.Println("DEBUG: committed:", m.Topic, string(m.Value))
						}
					}
				}
			}
		}
	}()
	return nil
}

var UseFunctionWithTimeoutError = errors.New("handler timeout")

func useFunctionWithTimeout(f func() error, timeout time.Duration) error {
	result := make(chan error, 1)
	go func() {
		result <- f()
	}()
	timer := time.NewTimer(1 * time.Minute)
	select {
	case <-timer.C:
		return UseFunctionWithTimeoutError
	case r := <-result:
		if !timer.Stop() {
			<-timer.C //drain timer channel for gc
		}
		return r
	}
}

func retry(f func() error, waitProvider func(n int64) time.Duration, timeout time.Duration) (err error) {
	err = errors.New("")
	start := time.Now()
	for i := int64(1); err != nil && time.Since(start) < timeout; i++ {
		err = useFunctionWithTimeout(f, 1*time.Minute)
		if errors.Is(err, UseFunctionWithTimeoutError) {
			return err
		}
		if err != nil {
			log.Println("ERROR: kafka listener error:", err)
			wait := waitProvider(i)
			if time.Since(start)+wait < timeout {
				log.Println("ERROR: retry after:", wait.String())
				time.Sleep(wait)
			} else {
				return err
			}
		}
	}
	return err
}
