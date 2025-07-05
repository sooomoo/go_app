package rmq

import (
	"errors"
	"fmt"
	"goapp/pkg/core"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	errClosed       error = errors.New("close normal")
	errNotConnected error = errors.New("not connect")
)

type Client struct {
	addr       string
	mutex      sync.RWMutex
	connection *amqp.Connection

	chNotifyConnClose chan *amqp.Error
	chCloseNormal     chan core.Empty
	queues            map[string]*Queue
}

const (
	resendDelay    = 1 * time.Second
	reconnectDelay = 1 * time.Second
)

func NewClient(addr string) *Client {
	return &Client{
		addr:              addr,
		mutex:             sync.RWMutex{},
		chNotifyConnClose: make(chan *amqp.Error, 1),
		chCloseNormal:     make(chan core.Empty),
		queues:            make(map[string]*Queue),
	}
}

func (c *Client) Connect() error {
	if c.connection != nil {
		return nil
	}

	if err := c.init(); err != nil {
		return err
	}
	c.autoReconnect()
	return nil
}

func (c *Client) NewQueue(name string) (*Queue, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if q, ok := c.queues[name]; ok {
		return q, nil
	}

	queue := newQueue(name)
	c.queues[name] = queue
	err := queue.open(c.connection)
	return queue, err
}

func (c *Client) CloseQueue(name string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	queue, ok := c.queues[name]
	if !ok {
		return
	}

	delete(c.queues, name)
	queue.close()
}

func (c *Client) autoReconnect() {
	go func() {
		for {
			select {
			case <-c.chCloseNormal:
				fmt.Println("closed normally")
				return
			case <-c.chNotifyConnClose:
				fmt.Println("connection closed. Reconnecting...")
				time.Sleep(reconnectDelay)

				if err := c.init(); err != nil {
					fmt.Println("connection reconnect failed.")
					continue
				}
			}
		}
	}()
}

func (c *Client) init() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.connection = nil
	close(c.chNotifyConnClose)
	c.chNotifyConnClose = make(chan *amqp.Error, 1)

	conn, err := amqp.Dial(c.addr)
	if err != nil {
		return err
	}
	conn.NotifyClose(c.chNotifyConnClose)
	c.connection = conn

	// 对已有的队列重新初始化
	for _, queue := range c.queues {
		queue.open(conn)
	}

	return nil
}

func (c *Client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.chCloseNormal <- core.Empty{}
	time.Sleep(100 * time.Millisecond)
	close(c.chCloseNormal)
	close(c.chNotifyConnClose)

	for _, queue := range c.queues {
		queue.close()
	}
	clear(c.queues)
	if err := c.connection.Close(); err != nil {
		return err
	}
	return nil
}
