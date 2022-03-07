package main

import (
	"context"
	"fmt"
	"github.com/gobackpack/rmq"
	"github.com/sirupsen/logrus"
	"sync"
)

func main() {
	// connect
	cred := rmq.NewCredentials()
	hub := rmq.NewHub(cred)

	hubCtx, hubCancel := context.WithCancel(context.Background())

	if err := hub.Connect(hubCtx, true); err != nil {
		logrus.Fatal(err)
	}

	// setup
	conf := rmq.NewConfig()
	conf.Exchange = "test_exchange_a"
	conf.Queue = "test_queue_a"
	conf.RoutingKey = "test_queue_a"

	if err := hub.CreateQueue(conf); err != nil {
		logrus.Fatal(err)
	}

	confB := rmq.NewConfig()
	confB.Exchange = "test_exchange_b"
	confB.Queue = "test_queue_b"
	confB.RoutingKey = "test_queue_b"

	if err := hub.CreateQueue(confB); err != nil {
		logrus.Fatal(err)
	}

	// listen for errors
	go func(ctx context.Context) {
		for {
			select {
			case err := <-hub.OnError:
				logrus.Error(err)
				break
			case <-ctx.Done():
				return
			}
		}
	}(hubCtx)

	// publish
	wg := sync.WaitGroup{}
	wg.Add(200)
	for i := 0; i < 100; i++ {
		go func(wg *sync.WaitGroup, i int, conf *rmq.Config) {
			hub.Publish(conf, []byte(fmt.Sprintf("queue_a - %d", i)))
			wg.Done()
		}(&wg, i, conf)

		go func(wg *sync.WaitGroup, i int, conf *rmq.Config) {
			hub.Publish(conf, []byte(fmt.Sprintf("queue_b - %d", i)))
			wg.Done()
		}(&wg, i, confB)
	}

	wg.Wait()
	hubCancel()
}
