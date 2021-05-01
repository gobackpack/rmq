package main

import (
	"github.com/gobackpack/rmq"
	"github.com/sirupsen/logrus"
	"sync"
)

func main() {
	credentials := rmq.NewCredentials()

	// config
	config := rmq.NewConfig()
	config.Exchange = "test_exchange_a"
	config.Queue = "test_queue_a"
	config.RoutingKey = "test_queue_a"

	// setup connection
	publisher := &rmq.Connection{
		Credentials: credentials,
		Config:      config,
		ResetSignal: make(chan int),
	}

	// pass true if there is only one publisher config
	// else manually call publisher.ApplyConfig(*Config) for each configuration (declare queues) and
	// call publisher.PublishWithConfig(*Config) if publisher.Config was not set!
	if err := publisher.Connect(true); err != nil {
		logrus.Fatal(err)
	}

	// optionally ListenNotifyClose and HandleResetSignalPublisher
	done := make(chan bool)

	go publisher.ListenNotifyClose(done)

	go publisher.HandleResetSignalPublisher(done)

	configB := rmq.NewConfig()
	configB.Exchange = "test_exchange_b"
	configB.Queue = "test_queue_b"
	configB.RoutingKey = "test_queue_b"

	if err := publisher.ApplyConfig(configB); err != nil {
		logrus.Error(err)
		return
	}

	configC := rmq.NewConfig()
	configC.Exchange = "test_exchange_c"
	configC.Queue = "test_queue_c"
	configC.RoutingKey = "test_queue_c"

	if err := publisher.ApplyConfig(configC); err != nil {
		logrus.Error(err)
		return
	}

	wg := sync.WaitGroup{}

	for i := 0; i < 2000; i++ {
		// this is not atomic operation
		// we need to keep it out of these goroutines
		wg.Add(3)

		go func() {
			defer wg.Done()

			// publish to default config exchange/queue
			if err := publisher.Publish([]byte("msg 1")); err != nil {
				logrus.Error(err)
			}
		}()

		go func() {
			defer wg.Done()

			// publish to configB exchange/queue
			if err := publisher.PublishWithConfig(configB, []byte("msg 2")); err != nil {
				logrus.Error(err)
			}
		}()

		go func() {
			defer wg.Done()

			// publish to configC exchange/queue
			if err := publisher.PublishWithConfig(configC, []byte("msg 3")); err != nil {
				logrus.Error(err)
			}
		}()
	}

	wg.Wait()

	close(done)

	<-done

	logrus.Info("rmq publisher sent messages")
}
