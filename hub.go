package rmq

import (
	"context"
)

type Hub struct {
	conn      *connection
	publisher chan *frame
	consumer  chan []byte
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
		conn:      newConnection(cred),
		publisher: make(chan *frame),
		consumer:  make(chan []byte),
	}
}

func (hub *Hub) Connect() error {
	if err := hub.conn.connect(); err != nil {
		return err
	}

	return hub.CreateChannel()
}

func (hub *Hub) CreateChannel() error {
	return hub.conn.createChannel()
}

func (hub *Hub) CreateQueue(conf *Config) error {
	return hub.conn.createQueue(conf)
}

func (hub *Hub) StartConsumer(ctx context.Context, conf *Config) (chan bool, chan []byte, chan error) {
	finished := make(chan bool)
	onMessage := make(chan []byte)
	onError := make(chan error)

	go func(ctx context.Context) {
		defer func() {
			finished <- true
		}()

		message, consErr := hub.conn.consume(conf)
		if consErr != nil {
			onError <- consErr
			return
		}

		for {
			select {
			case msg := <-message:
				onMessage <- msg.Body
				break
			case <-ctx.Done():
				if err := hub.conn.channel.Close(); err != nil {
					onError <- err
				}

				if err := hub.conn.amqpConn.Close(); err != nil {
					onError <- err
				}

				return
			}
		}
	}(ctx)

	return finished, onMessage, onError
}

func (hub *Hub) StartPublisher(ctx context.Context) chan error {
	onError := make(chan error)

	go func(ctx context.Context) {
		for {
			select {
			case fr := <-hub.publisher:
				if err := hub.conn.publish(fr.conf, fr.payload); err != nil {
					onError <- err
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	return onError
}

func (hub *Hub) Publish(conf *Config, payload []byte) {
	hub.publisher <- &frame{
		conf:    conf,
		payload: payload,
	}
}
