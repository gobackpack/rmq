![alt Go](https://img.shields.io/github/go-mod/go-version/gobackpack/rmq)

## ENV variables

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

## Usage

### Consumer

```go
// connect
cred := rmq.NewCredentials()
hub := rmq.NewHub(cred)

hubCtx, hubCancel := context.WithCancel(context.Background())
defer hubCancel()

if err := hub.Connect(hubCtx, false); err != nil {
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

// consumer
onMessage := make(chan []byte)
onError := make(chan error)

// listen for messages and errors
go func(ctx context.Context) {
    count := 0
    for {
        select {
        case msg := <-onMessage:
            count++
            logrus.Infof("[%d] - %s", count, msg)
            break
        case err := <-onError:
            logrus.Error(err)
            break
        case <-ctx.Done():
            return
        }
    }
}(hubCtx)

// consume
consumeFinished := hub.Consume(hubCtx, conf, onMessage, onError)

logrus.Info("listening for messages...")

<-consumeFinished
close(consumeFinished)
```


### Publisher

```go
// connect
cred := rmq.NewCredentials()
hub := rmq.NewHub(cred)

hubCtx, hubCancel := context.WithCancel(context.Background())

if err := hub.Connect(hubCtx, true); err != nil {
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

// listen for errors
go func(ctx context.Context) {
    for {
        select {
        case err := <-hub.OnPublishError:
            logrus.Error(err)
            break
        case <-ctx.Done():
            return
        }
    }
}(hubCtx)

// publish
wg := sync.WaitGroup{}
wg.Add(100)
for i := 0; i < 100; i++ {
    go func(wg *sync.WaitGroup, i int) {
        hub.Publish(conf, []byte(fmt.Sprintf("hello message %d", i)))
        wg.Done()
    }(&wg, i)
}

wg.Wait()
hubCancel()
```
