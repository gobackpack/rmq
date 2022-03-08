![alt Go](https://img.shields.io/github/go-mod/go-version/gobackpack/rmq)

### ENV variables

| ENV                | Default value |
|:-------------------|:-------------:|
| RMQ_HOST           | localhost     |
| RMQ_PORT           | 5672          |
| RMQ_USERNAME       | guest         |
| RMQ_PASSWORD       | guest         |
| RMQ_EXCHANGE       |               |
| RMQ_EXCHANGE_KIND  | direct        |
| RMQ_QUEUE          |               |
| RMQ_ROUTING_KEY    |               |
| RMQ_CONSUMER_TAG   |               |

### Usage

```go
// connect
cred := rmq.NewCredentials()
hub := rmq.NewHub(cred)

hubCtx, hubCancel := context.WithCancel(context.Background())
defer hubCancel()

if err := hub.Connect(); err != nil {
    logrus.Fatal(err)
}

// setup
conf := rmq.NewConfig()
conf.Exchange = "test_exchange_a"
conf.Queue = "test_queue_a"
conf.RoutingKey = "test_queue_a"

if err := hub.CreateQueue(conf); err != nil {
    logrus.Fatal(err)
}

confB := rmq.NewConfig()
confB.Exchange = "test_exchange_b"
confB.Queue = "test_queue_b"
confB.RoutingKey = "test_queue_b"

if err := hub.CreateQueue(confB); err != nil {
    logrus.Fatal(err)
}

// consume
// publish
```

* **Consumer:** [example/consumer](https://github.com/gobackpack/rmq/blob/main/example/consumer/main.go)
* **Publisher:** [example/publisher](https://github.com/gobackpack/rmq/blob/main/example/publisher/main.go)