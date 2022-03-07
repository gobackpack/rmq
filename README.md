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
cred := rmq.NewCredentials()
hub := rmq.NewHub(cred)

hubCtx, hubCancel := context.WithCancel(context.Background())
defer hubCancel()

if err := hub.Connect(hubCtx, false); err != nil {
    logrus.Fatal(err)
}

go func(ctx context.Context) {
    count := 0
    for {
        select {
        case err := <-hub.OnError:
            logrus.Error(err)
            break
        case msg := <-hub.OnMessage:
            count++
            logrus.Infof("[%d] - %s", count, msg)
            break
        case <-ctx.Done():
            return
        }
    }
}(hubCtx)

conf := rmq.NewConfig()
conf.Exchange = "test_exchange_a"
conf.Queue = "test_queue_a"
conf.RoutingKey = "test_queue_a"

if err := hub.CreateChannelQueue(conf); err != nil {
    logrus.Fatal(err)
}

consumeFinished := hub.Consume(hubCtx, conf)

logrus.Info("listening for messages...")

<-consumeFinished
close(consumeFinished)
```


### Publisher

```go
cred := rmq.NewCredentials()
hub := rmq.NewHub(cred)

hubCtx, hubCancel := context.WithCancel(context.Background())

if err := hub.Connect(hubCtx, true); err != nil {
    logrus.Fatal(err)
}

go func(ctx context.Context) {
    for {
        select {
        case err := <-hub.OnError:
            logrus.Error(err)
            break
        case <-ctx.Done():
            return
        }
    }
}(hubCtx)

conf := rmq.NewConfig()
conf.Exchange = "test_exchange_a"
conf.Queue = "test_queue_a"
conf.RoutingKey = "test_queue_a"

if err := hub.CreateChannelQueue(conf); err != nil {
    logrus.Fatal(err)
}

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
