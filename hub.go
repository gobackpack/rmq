package rmq

import (
	"context"
)

type Hub struct {
	OnError   chan error
	OnMessage chan []byte

	conn      *connection
	publisher chan *Frame
	consumer  chan []byte
}

type Frame struct {
	Conf    *Config
	Payload []byte
}

func NewHub(cred *Credentials) *Hub {
	if cred == nil {
		cred = NewCredentials()
	}

	return &Hub{
		OnError:   make(chan error),
		OnMessage: make(chan []byte),
		conn:      newConnection(cred),
		publisher: make(chan *Frame),
		consumer:  make(chan []byte),
	}
}

func (hub *Hub) Connect(ctx context.Context, publisher bool) error {
	if publisher {
		go hub.listenPublisher(ctx)
	}

	return hub.conn.connect()
}

func (hub *Hub) CreateChannel(conf *Config) error {
	return hub.conn.createChannel(conf)
}

func (hub *Hub) Publish(conf *Config, payload []byte) {
	hub.publisher <- &Frame{
		Conf:    conf,
		Payload: payload,
	}
}

func (hub *Hub) Consume(ctx context.Context, conf *Config) chan bool {
	finished := make(chan bool)

	go func(ctx context.Context) {
		defer func() {
			finished <- true
		}()

		message, consErr := hub.conn.consume(conf)
		if consErr != nil {
			hub.OnError <- consErr
			return
		}

		for {
			select {
			case msg := <-message:
				hub.OnMessage <- msg.Body
				break
			case <-ctx.Done():
				if err := hub.conn.Channel.Close(); err != nil {
					hub.OnError <- err
				}

				if err := hub.conn.AmqpConn.Close(); err != nil {
					hub.OnError <- err
				}

				return
			}
		}
	}(ctx)

	return finished
}

func (hub *Hub) listenPublisher(ctx context.Context) {
	for {
		select {
		case frame := <-hub.publisher:
			if err := hub.conn.publish(frame.Conf, frame.Payload); err != nil {
				if hub.OnError != nil {
					hub.OnError <- err
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
