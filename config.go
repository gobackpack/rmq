package rmq

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"os"
	"strings"
)

type Credentials struct {
	Host     string
	Port     string
	Username string
	Password string
}

type Config struct {
	Exchange     string
	ExchangeKind string

	Queue       string
	RoutingKey  string
	ConsumerTag string

	*Options
}

type Options struct {
	Exchange *ExchangeOpts
	QoS      *QoSOpts

	Queue     *QueueOpts
	QueueBind *QueueBindOpts

	Consume *ConsumeOpts
	Publish *PublishOpts
}

type ExchangeOpts struct {
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       amqp.Table
}

type QoSOpts struct {
	PrefetchCount int
	PrefetchSize  int
	Global        bool
}

type QueueOpts struct {
	Durable          bool
	DeleteWhenUnused bool
	Exclusive        bool
	Internal         bool
	NoWait           bool
	Args             amqp.Table
}

type QueueBindOpts struct {
	NoWait bool
	Args   amqp.Table
}

type ConsumeOpts struct {
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      amqp.Table
}

type PublishOpts struct {
	Mandatory bool
	Immediate bool
}

func NewCredentials() *Credentials {
	host := os.Getenv("RMQ_HOST")
	if strings.TrimSpace(host) == "" {
		host = "localhost"
	}

	port := os.Getenv("RMQ_PORT")
	if strings.TrimSpace(port) == "" {
		port = "5672"
	}

	username := os.Getenv("RMQ_USERNAME")
	if strings.TrimSpace(username) == "" {
		username = "guest"
	}

	password := os.Getenv("RMQ_PASSWORD")
	if strings.TrimSpace(password) == "" {
		password = "guest"
	}

	return &Credentials{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}
}

func NewConfig() *Config {
	exchange := os.Getenv("RMQ_EXCHANGE")
	exchangeKind := os.Getenv("RMQ_EXCHANGE_KIND")
	if strings.TrimSpace(exchangeKind) == "" {
		exchangeKind = "direct"
	}

	queue := os.Getenv("RMQ_QUEUE")
	routingKey := os.Getenv("RMQ_ROUTING_KEY")
	consumerTag := os.Getenv("RMQ_CONSUMER_TAG")

	return &Config{
		Exchange:     exchange,
		ExchangeKind: exchangeKind,

		Queue:       queue,
		RoutingKey:  routingKey,
		ConsumerTag: consumerTag,

		Options: &Options{
			Exchange: &ExchangeOpts{
				Durable:    true,
				AutoDelete: false,
				Internal:   false,
				NoWait:     false,
				Args:       nil,
			},
			QoS: &QoSOpts{
				PrefetchCount: 1,
				PrefetchSize:  0,
				Global:        false,
			},

			Queue: &QueueOpts{
				Durable:          true,
				DeleteWhenUnused: false,
				Exclusive:        false,
				NoWait:           false,
				Args:             nil,
			},
			QueueBind: &QueueBindOpts{
				NoWait: false,
				Args:   nil,
			},

			Consume: &ConsumeOpts{
				AutoAck:   true,
				Exclusive: false,
				NoLocal:   false,
				NoWait:    false,
				Args:      nil,
			},
			Publish: &PublishOpts{
				Mandatory: false,
				Immediate: false,
			},
		},
	}
}
