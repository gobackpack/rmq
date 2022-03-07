package rmq

import (
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"time"
)

type connection struct {
	Credentials   *Credentials
	ContentType   string
	ReconnectTime time.Duration

	// amqp
	AmqpConn *amqp.Connection
	Channel  *amqp.Channel
	Headers  amqp.Table
}

func newConnection(cred *Credentials) *connection {
	return &connection{
		Credentials:   cred,
		ContentType:   "text/plain",
		ReconnectTime: 20 * time.Second,
	}
}

func (conn *connection) connect() error {
	amqpConn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/",
		conn.Credentials.Username, conn.Credentials.Password, conn.Credentials.Host, conn.Credentials.Port))
	if err != nil {
		return err
	}
	conn.AmqpConn = amqpConn

	return nil
}

func (conn *connection) createChannel(conf *Config) error {
	if conn.AmqpConn == nil {
		return errors.New("amqp connection not initialized")
	}

	if conf == nil {
		return errors.New("invalid/nil Config")
	}

	amqpChannel, err := conn.AmqpConn.Channel()
	if err != nil {
		return err
	}

	conn.Channel = amqpChannel

	if err = conn.exchangeDeclare(conf.Exchange, conf.ExchangeKind, conf.Options.Exchange); err != nil {
		return err
	}

	if err = conn.qos(conf.Options.QoS); err != nil {
		return err
	}

	if _, err = conn.queueDeclare(conf.Queue, conf.Options.Queue); err != nil {
		return err
	}

	if err = conn.queueBind(conf.Queue, conf.RoutingKey, conf.Exchange, conf.Options.QueueBind); err != nil {
		return err
	}

	return nil
}

func (conn *connection) publish(conf *Config, payload []byte) error {
	err := conn.Channel.Publish(
		conf.Exchange,
		conf.RoutingKey,
		conf.Options.Publish.Mandatory,
		conf.Options.Publish.Immediate,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  conn.ContentType,
			Body:         payload,
			Headers:      conn.Headers,
		})

	return err
}

func (conn *connection) consume(conf *Config) (<-chan amqp.Delivery, error) {
	message, err := conn.Channel.Consume(
		conf.Queue,
		conf.ConsumerTag,
		conf.Options.Consume.AutoAck,
		conf.Options.Consume.Exclusive,
		conf.Options.Consume.NoLocal,
		conf.Options.Consume.NoWait,
		conf.Options.Consume.Args,
	)

	return message, err
}

func (conn *connection) queueDeclare(name string, opts *QueueOpts) (amqp.Queue, error) {
	queue, err := conn.Channel.QueueDeclare(
		name,
		opts.Durable,
		opts.DeleteWhenUnused,
		opts.Exclusive,
		opts.NoWait,
		opts.Args,
	)

	return queue, err
}

func (conn *connection) exchangeDeclare(name string, kind string, opts *ExchangeOpts) error {
	err := conn.Channel.ExchangeDeclare(
		name,
		kind,
		opts.Durable,
		opts.AutoDelete,
		opts.Internal,
		opts.NoWait,
		opts.Args,
	)

	return err
}

func (conn *connection) qos(opts *QoSOpts) error {
	err := conn.Channel.Qos(
		opts.PrefetchCount,
		opts.PrefetchSize,
		opts.Global,
	)

	return err
}

func (conn *connection) queueBind(queue string, routingKey string, exchange string, opts *QueueBindOpts) error {
	err := conn.Channel.QueueBind(
		queue,
		routingKey,
		exchange,
		opts.NoWait,
		opts.Args,
	)

	return err
}

// todo: finish implementation
func (conn *connection) listenNotifyClose(ctx context.Context) chan bool {
	cancelled := make(chan bool)
	connClose := make(chan *amqp.Error)
	conn.AmqpConn.NotifyClose(connClose)

	go func(ctx context.Context, connClose chan *amqp.Error) {
		for {
			select {
			case err := <-connClose:
				logrus.Warn("rmq connection lost: ", err)
				logrus.Warn("reconnecting to rmq in ", conn.ReconnectTime.String())

				time.Sleep(conn.ReconnectTime)

				if connErr := conn.connect(); connErr != nil {
					logrus.Fatal("failed to recreate rmq connection: ", connErr)
				}

				// important step!
				// recreate connClose channel so we can listen for NotifyClose once again
				connClose = make(chan *amqp.Error)
				conn.AmqpConn.NotifyClose(connClose)
				break
			case <-ctx.Done():
				return
			}
		}
	}(ctx, connClose)

	return cancelled
}
