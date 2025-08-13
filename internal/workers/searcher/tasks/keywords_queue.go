package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"goapp/internal/pkg"
	"goapp/internal/workers/searcher/stores/dao/model"
	"goapp/internal/workers/searcher/stores/dao/query"
	"goapp/pkg/db"
	"goapp/pkg/rmq"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gen/field"
)

type KeywordTask struct {
	mut           sync.RWMutex
	client        *rmq.Client
	keywordQueue  *rmq.Queue
	progressQueue *rmq.Queue
}

func NewKeywordTask(client *rmq.Client) *KeywordTask {
	q, err := client.NewQueue(string(pkg.MQTopicSearchKeywords))
	if err != nil {
		panic(err)
	}
	pq, err := client.NewQueue(string(pkg.MQTopicSearchKeywordsProgress))
	if err != nil {
		panic(err)
	}
	return &KeywordTask{
		mut:           sync.RWMutex{},
		client:        client,
		keywordQueue:  q,
		progressQueue: pq,
	}
}

func (k *KeywordTask) Run(ctx context.Context) error {
	msgs, err := k.keywordQueue.Consume(10)
	if err != nil {
		return err
	}

	go func(ctx context.Context, msgs <-chan amqp.Delivery) {
		for {
			select {
			case <-ctx.Done():
				k.mut.Lock()
				k.keywordQueue = nil
				k.progressQueue = nil
				k.mut.Unlock()

				k.client.CloseQueue(string(pkg.MQTopicSearchKeywords))
				k.client.CloseQueue(string(pkg.MQTopicSearchKeywordsProgress))
				return
			case msg := <-msgs:
				go k.handle(ctx, msg)
			}
		}
	}(ctx, msgs)

	return nil
}

func (k *KeywordTask) handle(ctx context.Context, msg amqp.Delivery) {
	if len(msg.Body) == 0 || msg.Type != "search_keyword" {
		return
	}
	var mp db.JSON
	if err := json.Unmarshal(msg.Body, &mp); err != nil {
		fmt.Println(err)
		return // log error
	}

	recordID := mp.GetInt64("record_id")
	fmt.Println(recordID)
	// 找到记录
	se := query.Q.TaskWebSearch
	record, err := se.WithContext(ctx).Where(se.ID.Eq(recordID)).First()
	recordTask := newTaskWebSearchRecord(recordID, record)
	if err != nil {
		recordTask.markAsError(ctx, err, 0)
		return
	}
	recordTask.maskAsRunning(ctx)
	// 开始搜索
	fmt.Println(record.Keywords)

	// report progress
	recordTask.maskAsFinish(ctx)
	k.reportProgress(recordID)
}

func (k *KeywordTask) reportProgress(recordID int64) {
	k.mut.Lock()
	defer k.mut.Unlock()
	if k.progressQueue == nil {
		return
	}

	data, err := json.Marshal(map[string]any{"record_id": recordID})
	if err != nil {
		fmt.Println(err)
		return // log error
	}
	err = k.progressQueue.Push(data)
	if err != nil {
		fmt.Println(err) // log error
	}
}

type TaskWebSearchRecord struct {
	recordID int64
	record   *model.TaskWebSearch
}

func newTaskWebSearchRecord(recordID int64, record *model.TaskWebSearch) *TaskWebSearchRecord {
	return &TaskWebSearchRecord{recordID: recordID, record: record}
}

func (t *TaskWebSearchRecord) markAsError(ctx context.Context, err error, progress float32) {
	se := query.Q.TaskWebSearch
	arr := []field.AssignExpr{
		se.Status.Value(uint8(pkg.SearcherStatusError)),
		se.StatusText.Value(err.Error()),
		se.UpdatedAt.Value(time.Now().Unix()),
	}
	if progress > 0 {
		arr = append(arr, se.Progress.Value(progress))
	}

	res, err := se.WithContext(ctx).Where(se.ID.Eq(t.recordID)).UpdateColumnSimple(arr...)
	if err != nil || res.RowsAffected <= 0 {
		// log error
		fmt.Println(err)
	}
}

func (t *TaskWebSearchRecord) maskAsRunning(ctx context.Context) {
	se := query.Q.TaskWebSearch
	res, err := se.WithContext(ctx).Where(se.ID.Eq(t.recordID)).UpdateColumnSimple(
		se.Status.Value(uint8(pkg.SearcherStatusRunning)),
		se.StatusText.Value("running"),
		se.SearchAt.Value(time.Now().Unix()),
		se.UpdatedAt.Value(time.Now().Unix()),
	)
	if err != nil || res.RowsAffected <= 0 {
		// log error
		fmt.Println(err)
	}
}

func (t *TaskWebSearchRecord) maskAsFinish(ctx context.Context) {
	se := query.Q.TaskWebSearch
	res, err := se.WithContext(ctx).Where(se.ID.Eq(t.recordID)).UpdateColumnSimple(
		se.Status.Value(uint8(pkg.SearcherStatusFinish)),
		se.StatusText.Value("finish"),
		se.FinishAt.Value(time.Now().Unix()),
		se.UpdatedAt.Value(time.Now().Unix()),
		se.Progress.Value(1),
	)
	if err != nil || res.RowsAffected <= 0 {
		// log error
		fmt.Println(err)
	}
}
