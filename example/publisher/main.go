package main

import (
	"context"
	"fmt"
	"github.com/gobackpack/rmq"
	"github.com/sirupsen/logrus"
	"sync"
)

func main() {
	cred := rmq.NewCredentials()
	hub := rmq.NewHub(cred)

	hubCtx, hubCancel := context.WithCancel(context.Background())

	if err := hub.Connect(hubCtx, true); err != nil {
		logrus.Fatal(err)
	}

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

	conf := rmq.NewConfig()
	conf.Exchange = "test_exchange_a"
	conf.Queue = "test_queue_a"
	conf.RoutingKey = "test_queue_a"

	if err := hub.CreateChannelQueue(conf); err != nil {
		logrus.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(wg *sync.WaitGroup, i int) {
			hub.Publish(conf, []byte(fmt.Sprintf("hello message %d", i)))
			wg.Done()
		}(&wg, i)
	}

	wg.Wait()
	hubCancel()
}
