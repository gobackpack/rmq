package main

import (
	"github.com/gobackpack/rmq"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

func main() {
	credentials := rmq.NewCredentials()

	// config
	config := rmq.NewConfig()
	config.Exchange = "test_exchange_b"
	config.Queue = "test_queue_b"
	config.RoutingKey = "test_queue_b	"

	// setup connection
	consumer := &rmq.Connection{
		Credentials: credentials,
		Config: config,
		HandleMsg: func(msg <-chan amqp.Delivery) {
			for m := range msg {
				logrus.Info(config.Queue + " - " + string(m.Body))
			}
		},
		ResetSignal: make(chan int),
	}

	if err := consumer.Connect(true); err != nil {
		logrus.Fatal(err)
	}

	// start consumer
	done := make(chan bool)

	// optionally ListenNotifyClose and HandleResetSignalConsumer
	go consumer.ListenNotifyClose(done)

	go consumer.HandleResetSignalConsumer(done)

	go func() {
		if err := consumer.Consume(done); err != nil {
			logrus.Error(err)
		}
	}()

	<-done
}
