package pkg_test

import (
	"fmt"
	"goapp/pkg/rmq"
	"testing"
	"time"
)

var mqClient *rmq.Client
var mqQueue *rmq.Queue

func init() {
}

func TestMain(m *testing.M) {
	if mqClient == nil {
		mqClient = rmq.NewClient("amqp://admin:admin@localhost:5672/")
		err := mqClient.Connect()
		if err != nil {
			panic(err)
		}
		mqQueue, err = mqClient.NewQueue("queue_test")
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("已经初始化过连接")
	}
	m.Run()
}

func TestPush(t *testing.T) {
	mqQueue.Push(fmt.Appendf(nil, "hello world at %s", time.Now().Format("2006-01-02 15:04:05")))
}
func TestConsume(t *testing.T) {
	deliveries, err := mqQueue.Consume(1)
	if err != nil {
		t.Error(err)
		return
	}
	for delivery := range deliveries {
		fmt.Println(string(delivery.Body))
		delivery.Ack(false)
	}
	t.Log("done")
}
