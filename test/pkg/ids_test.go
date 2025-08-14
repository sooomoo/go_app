package pkg_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"goapp/pkg/core"
	"goapp/pkg/ids"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

type Example struct {
	ID    int64   `json:"id,string"` // 增加 string 标签，序列化时转为字符串
	Name  string  `json:"name"`
	Value int64   `json:"value"` // 保持为数字
	Ptr   *int64  `json:"ptr"`
	Arr   []int64 `json:"arr"`
}

func TestInt64String(t *testing.T) {
	data := Example{
		ID:    1234567890123,
		Name:  "test",
		Value: 987654321,
		Arr:   []int64{1234567890123, 1234567890123},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(string(jsonData))

	var out Example
	err = json.Unmarshal(jsonData, &out)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(out)
}

func TestID(t *testing.T) {
	id := ids.NewID()
	fmt.Printf("New ID: %v\n", id)
	fmt.Printf("ID Time: %v, nodeid:%v, time back times:%v\n", ids.IDGetTimestamp(id), ids.IDGetNodeID(id), ids.IDGetClockBackwardTimes(id))
	seqID := ids.NewSeqID()
	fmt.Printf("New SeqID: %v\n", seqID)
	fmt.Printf("SeqID Hex: %s\n", seqID)
	fmt.Printf("SeqID B64: %s\n", seqID.Base64())
	fmt.Printf("SeqID Time: %v\n", seqID.Timestamp())
	var nilSeq ids.SeqID
	fmt.Printf("Nil SeqEq: %v\n", nilSeq == ids.NilSeqID)

	uidv7, _ := uuid.NewV7()
	fmt.Printf("New UUID: %s\n", strings.ReplaceAll(uidv7.String(), "-", ""))
	fmt.Printf("New UUID Base64: %s\n", base64.RawURLEncoding.EncodeToString(uidv7[:]))

	var bigID ids.BigID
	fmt.Printf("New BigEQ: %v\n", bigID == ids.NilBigID)

	var seqIDCounter uint32 = 0xffffffff
	seq := atomic.AddUint32(&seqIDCounter, 20)
	fmt.Println(seq)

	fmt.Println(seqID)
}

func TestSeqId(t *testing.T) {
	id := ids.NewSeqID()
	fmt.Printf("New SeqID: %v\n", id)
	fmt.Printf("SeqID Hex: %s\n", id)
	fmt.Printf("SeqID B64: %s\n", id.Base64())
	fmt.Printf("SeqID Time: %v\n", id.Timestamp())

	cnt := 10000
	wg := sync.WaitGroup{}
	mp := sync.Map{}
	wg.Add(cnt)
	for range cnt {
		go func() {
			defer wg.Done()
			for range 10000 {
				id := ids.NewSeqID()
				if _, loaded := mp.LoadOrStore(id, core.Empty{}); loaded {
					t.Errorf("Duplicate SeqID found: %v", id)
					return
				}
			}
		}()
	}
	wg.Wait()
	fmt.Printf("Set size after adding %d SeqIDs: %d\n", cnt, 0)
}

type SeqIDExample struct {
	ID  ids.SeqID   `json:"id"`
	Arr []ids.SeqID `json:"arr"`
	Age int         `json:"age"`
}

func TestSeqIDMarshalJsonExp(t *testing.T) {
	exp := SeqIDExample{
		ID: ids.NewSeqID(),
		Arr: []ids.SeqID{
			ids.NewSeqID(),
			ids.NewSeqID(),
			ids.NewSeqID(),
		},
		Age: 30,
	}
	data, err := json.Marshal(exp)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled SeqIDExample: %v\n", string(data))

	var out SeqIDExample
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled SeqIDExample: %v\n", out)
}

func TestSeqIDMarshalJson(t *testing.T) {
	id := ids.NewSeqID()
	data, err := json.Marshal(id)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled SeqID: %s\n", data)

	var out ids.SeqID
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled SeqID: %v\n", out)
}

func TestSeqIDMarshalTest(t *testing.T) {
	id := ids.NewSeqID()
	data, err := id.MarshalText()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled SeqID: %s\n", string(data))

	var out ids.SeqID
	err = out.UnmarshalText(data)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled SeqID: %v\n", out)
}

func BenchmarkSeqID(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ids.NewSeqID()
		}
	})
}

func TestNewID(t *testing.T) {
	id := ids.NewID()
	fmt.Printf("New ID: %d\n", id)

	mp := sync.Map{}
	cnt := 10000
	wg := sync.WaitGroup{}
	wg.Add(cnt)
	for range cnt {
		go func() {
			defer wg.Done()
			for range 10000 {
				id := ids.NewID()
				if _, loaded := mp.LoadOrStore(id, core.Empty{}); loaded {
					t.Errorf("Duplicate ID found: %v", id)
					return
				}
				times := ids.IDGetClockBackwardTimes(id)
				if times > 0 {
					fmt.Printf("New ID:%d, Clock back times:%v\n", id, times)
				}
			}
		}()
	}
	// 新开协程模拟时钟回退
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Second)
		delta := int64(15)
		ids.SetSnowIDNowMillisFunc(func() int64 {
			return time.Now().UnixMilli() - delta
		})
		time.Sleep(5 * time.Millisecond)
		delta = int64(30)
	}()
	wg.Wait()
	fmt.Printf("Set size after adding %d IDs: %d\n", cnt, 0)
}

func TestIDMany(t *testing.T) {
	for range 10 {
		id := ids.NewID()
		fmt.Printf("New RawID: %d, time: %v\n", id, ids.IDGetTimestamp(id))
	}
	for range 10 {
		id := ids.NewBigID()
		fmt.Printf("New BigID: %d, time: %v\n", id, id.Timestamp())
	}

	for range 10 {
		id := ids.NewSeqID()
		fmt.Printf("New SeqID: %s, time: %v\n", id, id.Timestamp())
	}
}

func BenchmarkBigID(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ids.NewID()
		}
	})
}

type BigIDExample struct {
	ID  int64   `json:"id"`
	Arr []int64 `json:"arr"`
	Age int     `json:"age"`
}

func TestBigIDMarshalJsonExp(t *testing.T) {
	exp := BigIDExample{
		ID: int64(ids.NewBigID()),
		Arr: []int64{
			int64(ids.NewBigID()),
			int64(ids.NewBigID()),
			int64(ids.NewBigID()),
		},
		Age: 30,
	}
	data, err := json.Marshal(exp)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled BigIDExample: %v\n", string(data))

	var out BigIDExample
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled BigIDExample: %v\n", out)
}

func TestBigIDMarshalJson(t *testing.T) {
	id := ids.NewBigID()
	data, err := json.Marshal(id)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled BigID: %s\n", data)

	var out ids.BigID
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled BigID: %v\n", out)
}

func TestBigIDMarshalTest(t *testing.T) {
	id := ids.NewBigID()
	data, err := id.MarshalText()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled BigID: %s\n", string(data))

	var out ids.BigID
	err = out.UnmarshalText(data)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled BigID: %v\n", out)
}

type UIDExample struct {
	ID  ids.UID   `json:"id"`
	Arr []ids.UID `json:"arr"`
	Age int       `json:"age"`
}

func TestUIDMarshalJsonExp(t *testing.T) {
	exp := UIDExample{
		ID: ids.NewUID(),
		Arr: []ids.UID{
			ids.NewUID(),
			ids.NewUID(),
			ids.NewUID(),
		},
		Age: 30,
	}
	data, err := json.Marshal(exp)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled UIDExample: %v\n", string(data))

	var out UIDExample
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled UIDExample: %v\n", out)
}

func TestUIDMarshalJson(t *testing.T) {
	id := ids.NewUID()
	data, err := json.Marshal(id)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled UID: %s\n", data)

	var out ids.UID
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled UID: %v\n", out)
}

func TestUIDMarshalTest(t *testing.T) {
	id := ids.NewUID()
	fmt.Println(id)
	data, err := id.MarshalText()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled UID: %s\n", string(data))

	id2, _ := ids.NewUIDFromHex(string(data))
	fmt.Printf("UID: %v \n", id2)

	var out ids.UID
	err = out.UnmarshalText(data)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled UID: %v\n", out)
}

func TestUIDBaseConvert(t *testing.T) {
	id := ids.NewUID()
	fmt.Printf("UID: %v, time: %v\n", id, id.TimeUnixMills())
	b64 := id.ToBase64()
	fmt.Printf("base:64, val: %s, len:%d\n", b64, len(b64))
	fB64, err := ids.NewUIDFromBase64(b64)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("from b64 uid: %v, time: %v\n", fB64, fB64.TimeUnixMills())
	if fB64 != id {
		fmt.Println("wrong", fB64)
	}
}

func BenchmarkUID(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ids.NewUID()
		}
	})
}

func TestMarshalJson(t *testing.T) {
	exp := UIDExample{
		ID: ids.NewUID(),
		Arr: []ids.UID{
			ids.NewUID(),
			ids.NewUID(),
			ids.NewUID(),
		},
		Age: 30,
	}
	data, err := json.Marshal(exp)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled UIDExample: %v\n", string(data))

	seqIDexp := SeqIDExample{
		ID: ids.NewSeqID(),
		Arr: []ids.SeqID{
			ids.NewSeqID(),
			ids.NewSeqID(),
			ids.NewSeqID(),
		},
		Age: 30,
	}
	seqdata, err := json.Marshal(seqIDexp)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled SeqIDExample: %v\n", string(seqdata))
	bigexp := BigIDExample{
		ID: int64(ids.NewBigID()),
		Arr: []int64{
			int64(ids.NewBigID()),
			int64(ids.NewBigID()),
			int64(ids.NewBigID()),
		},
		Age: 30,
	}
	bigdata, err := json.Marshal(bigexp)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled BigIDExample: %v\n", string(bigdata))
}
