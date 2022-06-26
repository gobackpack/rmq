package rmq

import (
	"context"
)

type Hub struct {
	conn *connection
}

type Consumer struct {
	Finished  chan bool
	OnMessage chan []byte
	OnError   chan error
}

type Publisher struct {
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

func (hub *Hub) StartConsumer(ctx context.Context, conf *Config) *Consumer {
	consumer := &Consumer{
		Finished:  make(chan bool),
		OnMessage: make(chan []byte),
		OnError:   make(chan error),
	}

	go func(ctx context.Context, consumer *Consumer) {
		defer func() {
			consumer.Finished <- true
		}()

		message, consErr := hub.conn.consume(conf)
		if consErr != nil {
			consumer.OnError <- consErr
			return
		}

		for {
			select {
			case msg := <-message:
				consumer.OnMessage <- msg.Body
				break
			case <-ctx.Done():
				if err := hub.conn.channel.Close(); err != nil {
					consumer.OnError <- err
				}

				if err := hub.conn.amqpConn.Close(); err != nil {
					consumer.OnError <- err
				}

				return
			}
		}
	}(ctx, consumer)

	return consumer
}

func (hub *Hub) StartPublisher(ctx context.Context, conf *Config) *Publisher {
	publisher := &Publisher{
		OnError: make(chan error),
		conf:    conf,
		publish: make(chan *frame),
	}

	go func(ctx context.Context, publisher *Publisher) {
		for {
			select {
			case fr := <-publisher.publish:
				if err := hub.conn.publish(fr.conf, fr.payload); err != nil {
					publisher.OnError <- err
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, publisher)

	return publisher
}

func (hub *Hub) Publish(payload []byte, publisher *Publisher) {
	publisher.publish <- &frame{
		conf:    publisher.conf,
		payload: payload,
	}
}
