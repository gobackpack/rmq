package rmq

import (
	"context"
	"github.com/sirupsen/logrus"
	"time"
)

type Hub struct {
	conn *connection
}

type Consumer struct {
	OnMessage chan []byte
	OnError   chan error
	Finished  chan bool
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

// Connect to RabbitMQ server. Listen for connection loss and attempt to reconnect.
// Signal will be sent to returned chan bool.
func (hub *Hub) Connect(ctx context.Context) (chan bool, error) {
	if err := hub.conn.connect(); err != nil {
		return nil, err
	}

	if err := hub.conn.createChannel(); err != nil {
		return nil, err
	}

	reconnected := hub.conn.listenNotifyClose(ctx)

	go func(ctx context.Context) {
		defer logrus.Warn("hub closed RabbitMQ connection")

		for {
			select {
			case <-ctx.Done():
				if err := hub.Close(); err != nil {
					logrus.Error(err)
				}

				return
			}
		}
	}(ctx)

	return reconnected, nil
}

func (hub *Hub) CreateQueue(conf *Config) error {
	return hub.conn.createQueue(conf)
}

// StartConsumer will create RabbitMQ consumer and listen for messages.
// Messages and errors are sent to OnMessage and OnError channels.
func (hub *Hub) StartConsumer(ctx context.Context, conf *Config) *Consumer {
	cons := &Consumer{
		OnMessage: make(chan []byte),
		OnError:   make(chan error),
		Finished:  make(chan bool),
	}

	go func(ctx context.Context, cons *Consumer) {
		defer close(cons.Finished)

		// listen for messages
		message, consErr := hub.conn.consume(conf)
		if consErr != nil {
			cons.OnError <- consErr
		}

		// handle messages
		for {
			select {
			case msg := <-message:
				if len(msg.Body) > 0 {
					cons.OnMessage <- msg.Body
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, cons)

	return cons
}

// CreatePublisher will create RabbitMQ publisher and private listener for messages to be published.
// All messages to be published are sent through private publish channel.
// Errors will be sent to OnError channel.
func (hub *Hub) CreatePublisher(ctx context.Context, conf *Config) *Publisher {
	pub := &Publisher{
		OnError: make(chan error),
		conf:    conf,
		publish: make(chan *frame),
	}

	// listen for messages to be published
	go func(ctx context.Context, pub *Publisher) {
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

// Publish message to RabbitMQ through private pub.publish channel.
// Thread-safe.
func (hub *Hub) Publish(payload []byte, pub *Publisher) {
	pub.publish <- &frame{
		conf:    pub.conf,
		payload: payload,
	}
}

// ReconnectTime can be used to override default reconnectTime for RabbitMQ connection.
func (hub *Hub) ReconnectTime(t time.Duration) {
	hub.conn.reconnectTime = t
}

// Close RabbitMQ connection
func (hub *Hub) Close() error {
	if hub.conn.amqpConn.IsClosed() {
		return nil
	}

	if err := hub.conn.channel.Close(); err != nil {
		logrus.Error(err)
	}

	if err := hub.conn.amqpConn.Close(); err != nil {
		return err
	}

	return nil
}
