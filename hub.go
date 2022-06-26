package rmq

import (
	"context"
	"github.com/sirupsen/logrus"
)

type Hub struct {
	conn *connection
}

type consumer struct {
	Finished  chan bool
	OnMessage chan []byte
	OnError   chan error
}

type publisher struct {
	OnError chan error
	conf    *Config
	publish chan *frame
}

type frame struct {
	conf    *Config
	payload []byte
}

func NewHub(cred *Credentials) *Hub {
	if cred == nil {
		cred = NewCredentials()
	}

	return &Hub{
		conn: newConnection(cred),
	}
}

func (hub *Hub) Connect() error {
	if err := hub.conn.connect(); err != nil {
		return err
	}

	return hub.conn.createChannel()
}

func (hub *Hub) CreateQueue(conf *Config) error {
	return hub.conn.createQueue(conf)
}

func (hub *Hub) StartConsumer(ctx context.Context, conf *Config) *consumer {
	cons := &consumer{
		Finished:  make(chan bool),
		OnMessage: make(chan []byte),
		OnError:   make(chan error),
	}

	go func(ctx context.Context, cons *consumer) {
		defer func() {
			close(cons.Finished)
		}()

		message, consErr := hub.conn.consume(conf)
		if consErr != nil {
			logrus.Error(consErr)
			return
		}

		for {
			select {
			case msg := <-message:
				cons.OnMessage <- msg.Body
			case <-ctx.Done():
				if err := hub.conn.channel.Close(); err != nil {
					cons.OnError <- err
				}

				if err := hub.conn.amqpConn.Close(); err != nil {
					cons.OnError <- err
				}

				return
			}
		}
	}(ctx, cons)

	return cons
}

func (hub *Hub) CreatePublisher(ctx context.Context, conf *Config) *publisher {
	pub := &publisher{
		OnError: make(chan error),
		conf:    conf,
		publish: make(chan *frame),
	}

	go func(ctx context.Context, pub *publisher) {
		for {
			select {
			case fr := <-pub.publish:
				if err := hub.conn.publish(fr.conf, fr.payload); err != nil {
					pub.OnError <- err
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, pub)

	return pub
}

func (hub *Hub) Publish(payload []byte, publisher *publisher) {
	publisher.publish <- &frame{
		conf:    publisher.conf,
		payload: payload,
	}
}
