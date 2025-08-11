package rmq

import (
	"context"
	"errors"
	"fmt"
	"goapp/pkg/core"
	"goapp/pkg/ids"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Queue struct {
	mutex sync.RWMutex
	name  string

	connection *amqp.Connection
	channel    *amqp.Channel
	queue      amqp.Queue

	chNotifyClose   chan *amqp.Error
	chNotifyConfirm chan amqp.Confirmation
	chReconnected   chan core.Empty

	connected atomic.Bool

	cancelComsume       context.CancelFunc
	cancelAutoReconnect context.CancelFunc
}

func newQueue(name string) *Queue {
	return &Queue{
		name:            name,
		mutex:           sync.RWMutex{},
		connected:       atomic.Bool{},
		chNotifyClose:   make(chan *amqp.Error, 1),
		chNotifyConfirm: make(chan amqp.Confirmation, 1),
		chReconnected:   make(chan core.Empty, 1),
	}
}

func (c *Queue) open(connection *amqp.Connection) error {
	c.connection = connection
	if err := c.init(); err != nil {
		return err
	}

	c.autoReconnect()
	return nil
}

func (c *Queue) autoReconnect() {
	if c.cancelAutoReconnect != nil {
		c.cancelAutoReconnect()
		time.Sleep(500 * time.Millisecond)
		c.cancelAutoReconnect = nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelAutoReconnect = cancel

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("channel cancelled.")
				return
			case <-c.chNotifyClose:
				fmt.Println("channel closed. Reconnecting...")
				c.connected.Store(false)

				time.Sleep(reconnectDelay)
				if err := c.init(); err != nil {
					fmt.Println("reconnect failed:", err)
					continue
				} else {
					c.chReconnected <- core.Empty{}
				}
			}
		}
	}(ctx)
}

func (c *Queue) init() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connection == nil {
		return errors.New("connection is nil")
	}

	c.connected.Store(false)
	close(c.chNotifyClose)
	close(c.chNotifyConfirm)

	c.chNotifyClose = make(chan *amqp.Error, 1)
	c.chNotifyConfirm = make(chan amqp.Confirmation, 1)

	var channel *amqp.Channel
	var err error

	if channel, err = c.connection.Channel(); err != nil {
		return err
	}
	if err = channel.Confirm(false); err != nil {
		return err
	}
	queue, err := channel.QueueDeclare(
		c.name,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	channel.NotifyClose(c.chNotifyClose)
	channel.NotifyPublish(c.chNotifyConfirm)

	c.channel = channel
	c.queue = queue
	c.connected.Store(true)
	return err
}

type messageOption struct {
	msgID   string
	msgType string
	userID  string
	appID   string

	retryTimes int
}

type optionFunc func(*messageOption)

func WithMsgID(msgID string) optionFunc {
	return func(option *messageOption) {
		option.msgID = msgID
	}
}
func WithMsgType(msgType string) optionFunc {
	return func(option *messageOption) {
		option.msgType = msgType
	}
}
func WithUserID(userID string) optionFunc {
	return func(option *messageOption) {
		option.userID = userID
	}
}
func WithAppID(appID string) optionFunc {
	return func(option *messageOption) {
		option.appID = appID
	}
}
func WithRetry(retryTimes int) optionFunc {
	if retryTimes < 1 {
		retryTimes = 1
	}
	return func(option *messageOption) {
		option.retryTimes = retryTimes
	}
}

func (c *Queue) Push(data []byte, options ...optionFunc) error {
	option := &messageOption{}
	for _, optionFunc := range options {
		optionFunc(option)
	}
	if len(option.msgID) == 0 {
		option.msgID = ids.NewUUID()
	}
	if option.retryTimes < 1 {
		option.retryTimes = 1
	}

	var err error
	for range option.retryTimes {
		if err = c.internalPush(data, option); err != nil {
			fmt.Println("push failed. Retrying...")
			time.Sleep(resendDelay)
			continue
		}
		confirm := <-c.chNotifyConfirm
		if confirm.Ack {
			fmt.Printf("push confirmed [%d]", confirm.DeliveryTag)
			return nil
		}
	}
	return err
}

func (c *Queue) internalPush(data []byte, option *messageOption) error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.connected.Load() {
		return errNotConnected
	}

	return c.channel.Publish(
		"",     // Exchange
		c.name, // Routing key
		false,  // Mandatory
		false,  // Immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         data,
			MessageId:    option.msgID,
			Timestamp:    time.Now(),
			Type:         option.msgType,
			UserId:       option.userID,
			AppId:        option.appID,
		},
	)
}

func (c *Queue) Consume(prefetchCount int) (<-chan amqp.Delivery, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.connected.Load() {
		return nil, errNotConnected
	}

	if prefetchCount < 1 {
		prefetchCount = 1
	}
	err := c.channel.Qos(
		prefetchCount, // prefetch count 表示在消费者发送 ack 确认之前，RabbitMQ 允许同一消费者最多接收多少条未确认的消息
		0,             // prefetch size 以字节为单位，指定消费者在发送 ack 之前可以接收的消息总大小。这里设置为 0，表示没有限制
		false,         // global 如果设置为 true，则 prefetch count 和 prefetch size 将对整个通道（所有消费者）生效。如果设置为 false，则仅对当前消费者生效
	)
	if err != nil {
		return nil, err
	}

	consumeFn := func() (<-chan amqp.Delivery, error) {
		return c.channel.Consume(
			c.name, // queue
			"",     // consumer
			false,  // auto-ack 手动确认
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			nil,    // args
		)
	}

	del, err := consumeFn()
	if err != nil {
		return nil, err
	}

	deliveries := make(chan amqp.Delivery)

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelComsume = cancel
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				close(deliveries)
				return // close consumer
			case d := <-del:
				deliveries <- d
			case <-c.chReconnected:
				for range 5 {
					del, err = consumeFn()
					if err != nil {
						time.Sleep(time.Second)
					} else {
						break
					}
				}
			}
		}
	}(ctx)
	return deliveries, nil
}

func (c *Queue) close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.connected.Store(false)

	if c.cancelComsume != nil {
		c.cancelComsume()
		c.cancelComsume = nil
	}
	if c.cancelAutoReconnect != nil {
		c.cancelAutoReconnect()
		c.cancelAutoReconnect = nil
	}
	time.Sleep(500 * time.Millisecond)

	close(c.chNotifyClose)
	close(c.chNotifyConfirm)
	close(c.chReconnected)

	if err := c.channel.Close(); err != nil {
		return err
	}
	c.channel = nil
	return nil
}
