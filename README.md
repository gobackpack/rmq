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


* **Consumer:** [example/consumer](https://github.com/gobackpack/rmq/blob/main/example/consumer/main.go)
* **Publisher:** [example/publisher](https://github.com/gobackpack/rmq/blob/main/example/publisher/main.go)