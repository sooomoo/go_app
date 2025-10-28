package pkg_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"goapp/pkg/core"
	"goapp/pkg/ids"
	"strings"
	"sync"
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

	uidv7, _ := uuid.NewV7()
	fmt.Printf("New UUID: %s\n", strings.ReplaceAll(uidv7.String(), "-", ""))
	fmt.Printf("New UUID Base64: %s\n", base64.RawURLEncoding.EncodeToString(uidv7[:]))

	var bigID ids.BigID
	fmt.Printf("New BigEQ: %v\n", bigID == ids.NilBigID)
}

func TestNewID(t *testing.T) {
	id := ids.NewBigID()
	fmt.Printf("New ID: %d\n", id)

	mp := sync.Map{}
	cnt := 10000
	wg := sync.WaitGroup{}
	wg.Add(cnt)
	for range cnt {
		go func() {
			defer wg.Done()
			for range 10000 {
				id := ids.NewBigID()
				if _, loaded := mp.LoadOrStore(id, core.Empty{}); loaded {
					t.Errorf("Duplicate ID found: %v", id)
					return
				}
				times := ids.IDGetClockBackwardTimes(int64(id))
				if times > 0 {
					fmt.Printf("New ID:%d, Clock back times:%v\n", id, times)
				}
			}
		}()
	}
	// 新开协程模拟时钟回退
	wg.Go(func() {
		time.Sleep(2 * time.Second)
		delta := int64(15)
		ids.SetSnowIDNowMillisFunc(func() int64 {
			return time.Now().UnixMilli() - delta
		})
		time.Sleep(5 * time.Millisecond)
		delta = int64(30)
	})
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
}

func BenchmarkBigID(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ids.NewBigID()
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

func TestUIDTimeParse(t *testing.T) {
	id1, err := ids.NewUIDFromHex("0198bb52-b106-7cb9-ba1a-49731af0e27f")
	id2, err := ids.NewUIDFromHex("0198bb5b-3509-7a19-94c2-a641d5490679")
	fmt.Println(err)
	ids := []ids.UID{
		id1,
		id2,
	}
	for _, i := range ids {
		fmt.Printf("time: %v\n", i.Time())
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
