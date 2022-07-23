package main

import (
	"context"
	"fmt"
	"github.com/gobackpack/rmq"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	// connect
	cred := rmq.NewCredentials()
	hub := rmq.NewHub(cred)

	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()

	reconnected, err := hub.Connect(hubCtx)
	if err != nil {
		logrus.Fatal(err)
	}

	// setup
	conf := rmq.NewConfig()
	conf.Exchange = "test_exchange_a"
	conf.Queue = "test_queue_a"
	conf.RoutingKey = "test_queue_a"

	if err = hub.CreateQueue(conf); err != nil {
		logrus.Fatal(err)
	}

	confB := rmq.NewConfig()
	confB.Exchange = "test_exchange_b"
	confB.Queue = "test_queue_b"
	confB.RoutingKey = "test_queue_b"

	if err = hub.CreateQueue(confB); err != nil {
		logrus.Fatal(err)
	}

	pub1 := hub.CreatePublisher(hubCtx, conf)
	pub2 := hub.CreatePublisher(hubCtx, confB)

	// listen for reconnection signal
	go func(ctx context.Context, hub *rmq.Hub) {
		defer logrus.Warn("reconnection signal listener finished")

		for {
			select {
			case _, ok := <-reconnected:
				if !ok {
					return
				}

				logrus.Info("reconnection signal received")

				if err = hub.CreateQueue(conf); err != nil {
					logrus.Fatal(err)
				}

				if err = hub.CreateQueue(confB); err != nil {
					logrus.Fatal(err)
				}

				logrus.Info("hub queue recreated")
			case <-hubCtx.Done():
				return
			}
		}
	}(hubCtx, hub)

	// listen for errors
	go func(ctx context.Context) {
		defer logrus.Warn("errors listener finished")

		for {
			select {
			case err = <-pub1.OnError:
				logrus.Error(err)
			case err = <-pub2.OnError:
				logrus.Error(err)
			case <-ctx.Done():
				return
			}
		}
	}(hubCtx)

	// publish
	wg := sync.WaitGroup{}
	delta := 3000
	wg.Add(delta * 2)
	for i := 0; i < delta; i++ {
		go func(wg *sync.WaitGroup, i int) {
			defer wg.Done()
			pub1.Publish([]byte(fmt.Sprintf("queue_a - %d", i)))
		}(&wg, i)

		go func(wg *sync.WaitGroup, i int) {
			defer wg.Done()
			pub2.Publish([]byte(fmt.Sprintf("queue_b - %d", i)))
		}(&wg, i)
	}

	wg.Wait()

	logrus.Warn("publisher finished...")

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
