package pkg_test

import (
	"encoding/json"
	"fmt"
	"goapp/pkg/core"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
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
	id := core.NewID()
	fmt.Printf("New ID: %v\n", id)
	fmt.Printf("ID Time: %v, nodeid:%v\n", core.IDTimestamp(id), core.IDNodeID(id))
	seqID := core.NewSeqID()
	fmt.Printf("New SeqID: %v\n", seqID)
	fmt.Printf("SeqID Hex: %s\n", seqID.Hex())
	fmt.Printf("SeqID B64: %s\n", seqID.Base64())
	fmt.Printf("SeqID Time: %v\n", seqID.Timestamp())
	var nilSeq core.SeqID
	fmt.Printf("Nil SeqEq: %v\n", nilSeq == core.NilSeqID)

	var bigID core.BigID
	fmt.Printf("New BigEQ: %v\n", bigID == core.NilBigID)

	var seqIDCounter uint32 = 0xffffffff
	seq := atomic.AddUint32(&seqIDCounter, 20)
	fmt.Println(seq)

	fmt.Println(seqID.Hex())
	r := core.NewCustomRadix34()
	fmt.Println(id, len(strconv.FormatInt(id, 10)))
	fmt.Println(r.Encode(int(id)))
	fmt.Println(strconv.FormatInt(id, 16))
}

func TestSeqId(t *testing.T) {
	id := core.NewSeqID()
	fmt.Printf("New SeqID: %v\n", id)
	fmt.Printf("SeqID Hex: %s\n", id.Hex())
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
				id := core.NewSeqID()
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

func BenchmarkSeqID(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			core.NewSeqID()
		}
	})
}

type SeqIDExample struct {
	ID  core.SeqID   `json:"id"`
	Arr []core.SeqID `json:"arr"`
	Age int          `json:"age"`
}

func TestSeqIDMarshalJsonExp(t *testing.T) {
	exp := SeqIDExample{
		ID: core.NewSeqID(),
		Arr: []core.SeqID{
			core.NewSeqID(),
			core.NewSeqID(),
			core.NewSeqID(),
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
	id := core.NewSeqID()
	data, err := json.Marshal(id)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled SeqID: %s\n", data)

	var out core.SeqID
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled SeqID: %v\n", out)
}

func TestSeqIDMarshalTest(t *testing.T) {
	id := core.NewSeqID()
	data, err := id.MarshalText()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled SeqID: %s\n", string(data))

	var out core.SeqID
	err = out.UnmarshalText(data)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled SeqID: %v\n", out)
}

func TestNewID(t *testing.T) {
	id := core.NewID()
	fmt.Printf("New BigID: %d\n", id)

	mp := sync.Map{}
	cnt := 10000
	wg := sync.WaitGroup{}
	wg.Add(cnt)
	for range cnt {
		go func() {
			defer wg.Done()
			for range 10000 {
				id := core.NewID()
				if _, loaded := mp.LoadOrStore(id, core.Empty{}); loaded {
					t.Errorf("Duplicate ID found: %v", id)
					return
				}
			}
		}()
	}
	wg.Wait()
	fmt.Printf("Set size after adding %d BigIDs: %d\n", cnt, 0)
}

func TestIDMany(t *testing.T) {
	for range 10 {
		id := core.NewID()
		fmt.Printf("New RawID: %d, time: %v\n", id, core.IDTimestamp(id))
	}
	for range 10 {
		id := core.NewBigID()
		fmt.Printf("New BigID: %d, time: %v\n", id, id.Timestamp())
	}

	for range 10 {
		id := core.NewSeqID()
		fmt.Printf("New SeqID: %s, time: %v\n", id.Hex(), id.Timestamp())
	}
}

func BenchmarkBigID(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			core.NewID()
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
		ID: int64(core.NewBigID()),
		Arr: []int64{
			int64(core.NewBigID()),
			int64(core.NewBigID()),
			int64(core.NewBigID()),
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
	id := core.NewBigID()
	data, err := json.Marshal(id)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled BigID: %s\n", data)

	var out core.BigID
	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled BigID: %v\n", out)
}

func TestBigIDMarshalTest(t *testing.T) {
	id := core.NewBigID()
	data, err := id.MarshalText()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Marshaled BigID: %s\n", string(data))

	var out core.BigID
	err = out.UnmarshalText(data)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Unmarshaled BigID: %v\n", out)
}
