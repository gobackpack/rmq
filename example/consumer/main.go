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

	// consume
	consumeFinished, onMessageC1, onErrorC1 := hub.Consume(hubCtx, conf)
	consumeFinished2, onMessageC2, onErrorC2 := hub.Consume(hubCtx, confB)

	// listen for messages and errors
	go func(ctx context.Context) {
		c1 := 0
		c2 := 0
		for {
			select {
			case msg := <-onMessageC1:
				c1++
				logrus.Infof("[%d] - %s", c1, msg)
				break
			case err := <-onErrorC1:
				logrus.Error(err)
				break
			case msg := <-onMessageC2:
				c2++
				logrus.Infof("[%d] - %s", c2, msg)
				break
			case err := <-onErrorC2:
				logrus.Error(err)
				break
			case <-ctx.Done():
				return
			}
		}
	}(hubCtx)

	logrus.Info("listening for messages...")

	<-consumeFinished
	close(consumeFinished)
	<-consumeFinished2
	close(consumeFinished2)
}
