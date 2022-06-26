package main

import (
	"context"
	"github.com/gobackpack/rmq"
	"github.com/sirupsen/logrus"
	"time"
)

func main() {
	// connect
	cred := rmq.NewCredentials()
	hub := rmq.NewHub(cred)
	hub.ReconnectTime(25 * time.Second)

	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()

	if err := hub.Connect(); err != nil {
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

	// consumer start
	consumer1 := hub.StartConsumer(hubCtx, conf)
	consumer2 := hub.StartConsumer(hubCtx, confB)

	// listen for messages and errors
	go func(ctx context.Context) {
		c1 := 0
		c2 := 0
		for {
			select {
			case msg := <-consumer1.OnMessage:
				c1++
				logrus.Infof("[%d] - %s", c1, msg)
			case err := <-consumer1.OnError:
				logrus.Error(err)
			case msg := <-consumer2.OnMessage:
				c2++
				logrus.Infof("[%d] - %s", c2, msg)
			case err := <-consumer2.OnError:
				logrus.Error(err)
			case <-ctx.Done():
				return
			}
		}
	}(hubCtx)

	logrus.Info("listening for messages...")

	<-consumer1.Finished
	<-consumer2.Finished
}
