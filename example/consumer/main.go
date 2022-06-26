package main

import (
	"context"
	"fmt"
	"github.com/gobackpack/rmq"
	"github.com/sirupsen/logrus"
	"time"
)

func main() {
	// connect
	cred := rmq.NewCredentials()
	hub := rmq.NewHub(cred)
	hub.ReconnectTime(30 * time.Second)

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

	// consumer start
	// consCtx will make sure previous invalid consumers get closed, prevents memory leak from consumers that lost connection
	consCtx, consCancel := context.WithCancel(hubCtx)
	consumer1 := hub.StartConsumer(consCtx, conf)
	consumer2 := hub.StartConsumer(consCtx, confB)

	// listen for reconnection signal, recreate queue and start consumers again
	go func(hub *rmq.Hub, consumer1 *rmq.Consumer, consumer2 *rmq.Consumer) {
		consCounter := 0

		for {
			select {
			case <-reconnected:
				logrus.Info("reconnection signal received")
				consCounter++

				// close previous invalid message handlers from consumers that lost connection
				consCancel()
				// make sure to recreate consumer context so new consumers and message handlers can be closed properly too
				consCtx, consCancel = context.WithCancel(hubCtx)

				if err = hub.CreateQueue(conf); err != nil {
					logrus.Fatal(err)
				}

				if err = hub.CreateQueue(confB); err != nil {
					logrus.Fatal(err)
				}

				logrus.Info("hub queue recreated")

				consumer1 = hub.StartConsumer(consCtx, conf)
				consumer2 = hub.StartConsumer(consCtx, confB)

				// listen for messages and errors
				go handleConsumerMessages(consCtx, consumer1, fmt.Sprintf("consumer 1 child #%d", consCounter))
				go handleConsumerMessages(consCtx, consumer2, fmt.Sprintf("consumer 2 child #%d", consCounter))

				logrus.Info("listening for messages...")
			}
		}
	}(hub, consumer1, consumer2)

	// listen for messages and errors
	// passing consCtx to handlers will make sure they get successfully closed when reconnected signal occurs,
	// so they do not hang in memory
	go handleConsumerMessages(consCtx, consumer1, "parent consumer 1")
	go handleConsumerMessages(consCtx, consumer2, "parent consumer 2")

	logrus.Info("listening for messages...")

	for {
	}
}

func handleConsumerMessages(ctx context.Context, cons *rmq.Consumer, name string) {
	logrus.Infof("%s started", name)

	defer func() {
		logrus.Warnf("%s closed", name)
	}()

	c := 0
	for {
		select {
		case msg := <-cons.OnMessage:
			c++
			logrus.Infof("[%d] - %s", c, msg)
		case err := <-cons.OnError:
			logrus.Error(err)
		case <-ctx.Done():
			return
		}
	}
}
