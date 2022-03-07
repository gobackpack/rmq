package main

import (
	"context"
	"github.com/gobackpack/rmq"
	"github.com/sirupsen/logrus"
)

func main() {
	// connect
	cred := rmq.NewCredentials()
	hub := rmq.NewHub(cred)

	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()

	if err := hub.Connect(hubCtx, false); err != nil {
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

	// listen for messages and errors
	go func(ctx context.Context) {
		count := 0
		for {
			select {
			case err := <-hub.OnError:
				logrus.Error(err)
				break
			case msg := <-hub.OnMessage:
				count++
				logrus.Infof("[%d] - %s", count, msg)
				break
			case <-ctx.Done():
				return
			}
		}
	}(hubCtx)

	// consume
	consumeFinished := hub.Consume(hubCtx, conf)
	consumeFinishedB := hub.Consume(hubCtx, confB)

	logrus.Info("listening for messages...")

	<-consumeFinished
	close(consumeFinished)
	<-consumeFinishedB
	close(consumeFinishedB)
}
