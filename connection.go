package rmq

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"os"
	"time"
)

const (
	Reconnected = iota
)

var (
	// reconnectTime is default time to wait for rmq reconnect on AmqpConn.NotifyClose() event - situation when rmq sends signal about shutdown
	reconnectTime = 20 * time.Second
)

// Connection for RMQ
type Connection struct {
	Credentials *Credentials
	Config      *Config

	// amqp
	AmqpConn    *amqp.Connection
	Channel     *amqp.Channel
	Headers     amqp.Table
	ContentType string

	// connection reset
	ResetSignal   chan int
	ReconnectTime time.Duration
	Retrying      bool

	// callbacks
	HandleMsg                  func(msg <-chan amqp.Delivery)
	HandleResetSignalConsumer  func(chan bool)
	HandleResetSignalPublisher func(chan bool)
}

// Connect to RabbitMQ and initialize channel
func (conn *Connection) Connect(applyConfig bool) error {
	if conn.Credentials == nil {
		return errors.New("invalid/nil Credentials")
	}

	conn.applyDefaults()

	amqpConn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", conn.Credentials.Username, conn.Credentials.Password, conn.Credentials.Host, conn.Credentials.Port))
	if err != nil {
		return err
	}
	conn.AmqpConn = amqpConn

	if applyConfig {
		if err = conn.ApplyConfig(conn.Config); err != nil {
			return err
		}
	}

	return nil
}

// ApplyConfig will initialize channel, exchange, qos and bind queues
// RabbitMQ declarations
func (conn *Connection) ApplyConfig(config *Config) error {
	if conn.AmqpConn == nil {
		return errors.New("amqp connection not initialized")
	}

	if config == nil {
		return errors.New("invalid/nil Config")
	}

	amqpChannel, err := conn.AmqpConn.Channel()
	if err != nil {
		return err
	}

	conn.Channel = amqpChannel

	if err = conn.exchangeDeclare(config.Exchange, config.ExchangeKind, config.Options.Exchange); err != nil {
		return err
	}

	if err = conn.qos(config.Options.QoS); err != nil {
		return err
	}

	if _, err = conn.queueDeclare(config.Queue, config.Options.Queue); err != nil {
		return err
	}

	if err = conn.queueBind(config.Queue, config.RoutingKey, config.Exchange, config.Options.QueueBind); err != nil {
		return err
	}

	return nil
}

// Consume data from RMQ
func (conn *Connection) Consume(done chan bool) error {
	msg, err := conn.Channel.Consume(
		conn.Config.Queue,
		conn.Config.ConsumerTag,
		conn.Config.Options.Consume.AutoAck,
		conn.Config.Options.Consume.Exclusive,
		conn.Config.Options.Consume.NoLocal,
		conn.Config.Options.Consume.NoWait,
		conn.Config.Options.Consume.Args,
	)
	if err != nil {
		return err
	}

	go conn.HandleMsg(msg)

	logrus.Info("waiting for messages...")

	for {
		select {
		case <-done:
			if err = conn.Channel.Close(); err != nil {
				logrus.Error("failed to close channel: ", err.Error())
				return err
			}

			if err = conn.AmqpConn.Close(); err != nil {
				logrus.Error("failed to close connection: ", err.Error())
				return err
			}

			return nil
		}
	}
}

// Publish payload to RMQ
func (conn *Connection) Publish(payload []byte) error {
	if conn.Config == nil {
		return errors.New("invalid/nil Config")
	}

	err := conn.Channel.Publish(
		conn.Config.Exchange,
		conn.Config.RoutingKey,
		conn.Config.Options.Publish.Mandatory,
		conn.Config.Options.Publish.Immediate,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  conn.ContentType,
			Body:         payload,
			Headers:      conn.Headers,
		})

	return err
}

func (conn *Connection) PublishWithConfig(config *Config, payload []byte) error {
	err := conn.Channel.Publish(
		config.Exchange,
		config.RoutingKey,
		config.Options.Publish.Mandatory,
		config.Options.Publish.Immediate,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  conn.ContentType,
			Body:         payload,
			Headers:      conn.Headers,
		})

	return err
}

// WithHeaders will set headers to be sent
func (conn *Connection) WithHeaders(h amqp.Table) *Connection {
	conn.Headers = h

	return conn
}

// ListenNotifyClose will listen for rmq connection shutdown and attempt to re-create rmq connection
func (conn *Connection) ListenNotifyClose(done chan bool) {
	connClose := make(chan *amqp.Error)
	conn.AmqpConn.NotifyClose(connClose)

	go func() {
		for {
			select {
			case err := <-connClose:
				logrus.Warn("rmq connection lost: ", err)
				logrus.Warn("reconnecting to rmq in ", conn.ReconnectTime.String())

				conn.Retrying = true

				time.Sleep(conn.ReconnectTime)

				if err := conn.recreateConn(); err != nil {
					killService("failed to recreate rmq connection: ", err)
				}

				logrus.Infof("sending signal %v to rmq connection", Reconnected)

				conn.ResetSignal <- Reconnected

				logrus.Infof("signal %v sent to rmq connection", Reconnected)

				// important step!
				// recreate connClose channel so we can listen for NotifyClose once again
				connClose = make(chan *amqp.Error)
				conn.AmqpConn.NotifyClose(connClose)

				conn.Retrying = false
			}
		}
	}()

	<-done
}

// queueDeclare is helper function to declare queue
func (conn *Connection) queueDeclare(name string, opts *QueueOpts) (amqp.Queue, error) {
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

// exchangeDeclare is helper function to declare exchange
func (conn *Connection) exchangeDeclare(name string, kind string, opts *ExchangeOpts) error {
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

// qos is helper function to define QoS for channel
func (conn *Connection) qos(opts *QoSOpts) error {
	err := conn.Channel.Qos(
		opts.PrefetchCount,
		opts.PrefetchSize,
		opts.Global,
	)

	return err
}

// queueBind is helper function to bind queue to exchange
func (conn *Connection) queueBind(queue string, routingKey string, exchange string, opts *QueueBindOpts) error {
	err := conn.Channel.QueueBind(
		queue,
		routingKey,
		exchange,
		opts.NoWait,
		opts.Args,
	)

	return err
}

// applyDefaults is helper function to setup some default Connection properties
func (conn *Connection) applyDefaults() {
	if conn.ReconnectTime == 0 {
		conn.ReconnectTime = reconnectTime
	}

	if conn.HandleResetSignalConsumer == nil {
		conn.HandleResetSignalConsumer = conn.handleResetSignalConsumer
	}

	if conn.HandleResetSignalPublisher == nil {
		conn.HandleResetSignalPublisher = conn.handleResetSignalPublisher
	}

	if conn.ContentType == "" {
		conn.ContentType = "text/plain"
	}
}

// handleResetSignalConsumer is default callback for consumer to run when reset signal occurs
func (conn *Connection) handleResetSignalConsumer(done chan bool) {
	go func(done chan bool) {
		for {
			select {
			case signal := <-conn.ResetSignal:
				logrus.Warn("consumer received rmq connection reset signal: ", signal)

				if done == nil {
					done = make(chan bool)
				}

				go func() {
					if err := conn.Consume(done); err != nil {
						logrus.Fatal("rmq failed to consume: ", err)
					}
				}()
			}
		}
	}(done)

	<-done
}

// handleResetSignalPublisher is default callback for publisher to run when reset signal occurs
func (conn *Connection) handleResetSignalPublisher(done chan bool) {
	go func() {
		for {
			select {
			case signal := <-conn.ResetSignal:
				logrus.Warn("publisher received rmq connection reset signal: ", signal)
			}
		}
	}()

	<-done
}

// recreateConn for rmq
func (conn *Connection) recreateConn() error {
	logrus.Info("trying to recreate rmq connection for host: ", conn.Credentials.Host)

	return conn.Connect(true)
}

// killService with message passed to console output
func killService(msg ...interface{}) {
	logrus.Warn(msg...)
	os.Exit(101)
}
