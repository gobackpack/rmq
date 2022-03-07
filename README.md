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

confB := rmq.NewConfig()
confB.Exchange = "test_exchange_b"
confB.Queue = "test_queue_b"
confB.RoutingKey = "test_queue_b"

if err := hub.CreateQueue(confB); err != nil {
    logrus.Fatal(err)
}

// consume
consumeFinished, onMessageC1, onErrorC1 := hub.Consume(hubCtx, conf)
consumeFinished2, onMessageC2, onErrorC2 := hub.Consume(hubCtx, confB)

// listen for messages and errors
go func(ctx context.Context) {
    c1 := 0
    c2 := 0
    for {
        select {
        case msg := <-onMessageC1:
            c1++
            logrus.Infof("[%d] - %s", c1, msg)
            break
        case err := <-onErrorC1:
            logrus.Error(err)
            break
        case msg := <-onMessageC2:
            c2++
            logrus.Infof("[%d] - %s", c2, msg)
            break
        case err := <-onErrorC2:
            logrus.Error(err)
            break
        case <-ctx.Done():
            return
        }
    }
}(hubCtx)

logrus.Info("listening for messages...")

<-consumeFinished
close(consumeFinished)
<-consumeFinished2
close(consumeFinished2)
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

confB := rmq.NewConfig()
confB.Exchange = "test_exchange_b"
confB.Queue = "test_queue_b"
confB.RoutingKey = "test_queue_b"

if err := hub.CreateQueue(confB); err != nil {
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
wg.Add(200)
for i := 0; i < 100; i++ {
    go func(wg *sync.WaitGroup, i int, conf *rmq.Config) {
        hub.Publish(conf, []byte(fmt.Sprintf("queue_a - %d", i)))
        wg.Done()
    }(&wg, i, conf)

    go func(wg *sync.WaitGroup, i int, conf *rmq.Config) {
        hub.Publish(conf, []byte(fmt.Sprintf("queue_b - %d", i)))
        wg.Done()
    }(&wg, i, confB)
}

wg.Wait()
hubCancel()
```
