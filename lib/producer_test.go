package lib

import (
	"context"
	k "github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"github.com/segmentio/kafka-go"
	"log"
	"sync"
	"testing"
	"time"
)

func TestProducerExperiment(t *testing.T) {
	t.Skip("experiment")
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	testCtx, testStop := context.WithCancel(context.Background())
	defer testStop()

	_, zkIp, err := Zookeeper(testCtx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	zkUrl := zkIp + ":2181"

	kafkaUrl, err := Kafka(testCtx, wg, zkUrl)
	if err != nil {
		t.Error(err)
		return
	}

	startSig := sync.WaitGroup{}
	startSig.Add(1)
	ctx, cancel := context.WithCancel(testCtx)

	producer1Counter := 0
	go func() {
		topic := "test1"
		err = k.InitTopic(kafkaUrl, topic)
		if err != nil {
			t.Error(err)
			return
		}
		producer := &kafka.Writer{
			Addr:        kafka.TCP(kafkaUrl),
			Topic:       topic,
			MaxAttempts: 10,
		}
		wg.Add(1)
		defer wg.Done()
		defer producer.Close()
		log.Println("producer 1 initialized")
		startSig.Wait()
		log.Println("producer 1 started")
		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := producer.WriteMessages(context.Background(), kafka.Message{
					Key:   []byte("foo"),
					Value: []byte("bar"),
					Time:  time.Now(),
				})
				if err != nil {
					t.Error(err)
				}
				producer1Counter++
			}
		}
	}()

	producer2Counter := 0
	go func() {
		topic := "test2"
		err = k.InitTopic(kafkaUrl, topic)
		if err != nil {
			t.Error(err)
			return
		}
		producer := &kafka.Writer{
			Addr:        kafka.TCP(kafkaUrl),
			Topic:       topic,
			MaxAttempts: 10,
			BatchSize:   1,
		}
		wg.Add(1)
		defer wg.Done()
		defer producer.Close()
		log.Println("producer 2 initialized")
		startSig.Wait()
		log.Println("producer 2 started")
		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := producer.WriteMessages(context.Background(), kafka.Message{
					Key:   []byte("foo"),
					Value: []byte("bar"),
					Time:  time.Now(),
				})
				if err != nil {
					t.Error(err)
				}
				producer2Counter++
			}
		}
	}()

	time.Sleep(1 * time.Second)

	startSig.Done()
	time.Sleep(10 * time.Second)
	cancel()

	t.Log(producer1Counter, producer2Counter)

}
