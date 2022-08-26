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
	"io/ioutil"
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
		Logger:         log.New(ioutil.Discard, "", 0),
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

func retry(f func() error, waitProvider func(n int64) time.Duration, timeout time.Duration) (err error) {
	err = errors.New("")
	start := time.Now()
	for i := int64(1); err != nil && time.Since(start) < timeout; i++ {
		err = f()
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
