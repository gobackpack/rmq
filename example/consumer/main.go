package main

import (
	"context"
	"github.com/gobackpack/rmq"
	"github.com/sirupsen/logrus"
)

func main() {
	cred := rmq.NewCredentials()
	hub := rmq.NewHub(cred)

	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()

	if err := hub.Connect(hubCtx, false); err != nil {
		logrus.Fatal(err)
	}

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

	conf := rmq.NewConfig()
	conf.Exchange = "test_exchange_a"
	conf.Queue = "test_queue_a"
	conf.RoutingKey = "test_queue_a"

	if err := hub.CreateChannel(conf); err != nil {
		logrus.Fatal(err)
	}

	finished := hub.Consume(hubCtx, conf)

	logrus.Info("listening for messages...")

	<-finished
}
