package rmq

import (
	"context"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"time"
)

type connection struct {
	credentials   *Credentials
	contentType   string
	reconnectTime time.Duration

	// amqp
	amqpConn *amqp.Connection
	channel  *amqp.Channel
	headers  amqp.Table
}

func newConnection(cred *Credentials) *connection {
	return &connection{
		credentials:   cred,
		contentType:   "text/plain",
		reconnectTime: 20 * time.Second,
	}
}

func (conn *connection) connect() error {
	amqpConn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/",
		conn.credentials.Username, conn.credentials.Password, conn.credentials.Host, conn.credentials.Port))
	if err != nil {
		return err
	}
	conn.amqpConn = amqpConn

	return nil
}

func (conn *connection) createChannel() error {
	amqpChannel, err := conn.amqpConn.Channel()
	if err != nil {
		return err
	}

	conn.channel = amqpChannel

	return nil
}

func (conn *connection) createQueue(conf *Config) error {
	if err := conn.exchangeDeclare(conf.Exchange, conf.ExchangeKind, conf.Options.Exchange); err != nil {
		return err
	}

	if err := conn.qos(conf.Options.QoS); err != nil {
		return err
	}

	if _, err := conn.queueDeclare(conf.Queue, conf.Options.Queue); err != nil {
		return err
	}

	if err := conn.queueBind(conf.Queue, conf.RoutingKey, conf.Exchange, conf.Options.QueueBind); err != nil {
		return err
	}

	return nil
}

func (conn *connection) publish(conf *Config, payload []byte) error {
	err := conn.channel.Publish(
		conf.Exchange,
		conf.RoutingKey,
		conf.Options.Publish.Mandatory,
		conf.Options.Publish.Immediate,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  conn.contentType,
			Body:         payload,
			Headers:      conn.headers,
		})

	return err
}

func (conn *connection) consume(conf *Config) (<-chan amqp.Delivery, error) {
	message, err := conn.channel.Consume(
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
	queue, err := conn.channel.QueueDeclare(
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
	err := conn.channel.ExchangeDeclare(
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
	err := conn.channel.Qos(
		opts.PrefetchCount,
		opts.PrefetchSize,
		opts.Global,
	)

	return err
}

func (conn *connection) queueBind(queue string, routingKey string, exchange string, opts *QueueBindOpts) error {
	err := conn.channel.QueueBind(
		queue,
		routingKey,
		exchange,
		opts.NoWait,
		opts.Args,
	)

	return err
}

// listenNotifyClose will listen for connection loss, attempt to reconnect and send signal after successful reconnection.
// todo: support different reconnection signals
func (conn *connection) listenNotifyClose(ctx context.Context) chan bool {
	reconnected := make(chan bool)
	connClose := make(chan *amqp.Error)
	conn.amqpConn.NotifyClose(connClose)

	go func(ctx context.Context, connClose chan *amqp.Error) {
		defer close(reconnected)

		for {
			select {
			case err := <-connClose:
				logrus.Warn("rmq connection lost: ", err)
				logrus.Warn("reconnecting to rmq in ", conn.reconnectTime.String())

				time.Sleep(conn.reconnectTime)

				if connErr := conn.connect(); connErr != nil {
					logrus.Fatal("failed to recreate rmq connection: ", connErr)
				}

				if connErr := conn.createChannel(); connErr != nil {
					logrus.Fatal("failed to recreate rmq channel: ", connErr)
				}

				logrus.Info("reconnected")

				// important step!
				// recreate connClose channel, so we can listen for NotifyClose once again
				connClose = make(chan *amqp.Error)
				conn.amqpConn.NotifyClose(connClose)

				logrus.Info("sending reconnection signal")
				reconnected <- true
			case <-ctx.Done():
				return
			}
		}
	}(ctx, connClose)

	return reconnected
}
