package pkg_test

import (
	"encoding/json"
	"fmt"
	"goapp/pkg/collection"
	"goapp/pkg/core"
	"os"
	"sync"
	"testing"
)

func init() {
	os.Setenv("node_id", "2")
}

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

func TestSeqId(t *testing.T) {
	id := core.NewSeqID()
	fmt.Printf("New SeqID: %v\n", id)
	fmt.Printf("SeqID Hex: %s\n", id.Hex())
	fmt.Printf("SeqID B64: %s\n", id.Base64())

	set := collection.Set[core.SeqID]{}
	cnt := 2000
	wg := sync.WaitGroup{}
	wg.Add(cnt)
	for range cnt {
		go func() {
			defer wg.Done()
			for range 10000 {
				id := core.NewSeqID()
				if set.Contains(id) {
					t.Errorf("Duplicate SeqID found: %v", id)
					return
				}
				set.Add(id)
			}
		}()
	}
	wg.Wait()
	fmt.Printf("Set size after adding %d SeqIDs: %d\n", cnt, set.Size())
}
func BenchmarkConcurrentNewSeqID(b *testing.B) {
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

func TestBigIdNew(t *testing.T) {
	id := core.NewBigID()
	fmt.Printf("New BigID: %d\n", id)

	set := collection.Set[core.BigID]{}
	cnt := 2000
	wg := sync.WaitGroup{}
	wg.Add(cnt)
	for range cnt {
		go func() {
			defer wg.Done()
			for range 10000 {
				id := core.NewBigID()
				if set.Contains(id) {
					t.Errorf("Duplicate BigID found: %v", id)
					return
				}
				set.Add(id)
			}
		}()
	}
	wg.Wait()
	fmt.Printf("Set size after adding %d BigIDs: %d\n", cnt, set.Size())
}
func TestBigIdNewMany(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(10)
	for range 10 {
		go func() {
			defer wg.Done()
			id := core.NewBigID()
			fmt.Printf("New BigID: %d\n", id)
		}()
	}
	wg.Wait()
}
func BenchmarkConcurrentID(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			core.NewBigID()
		}
	})
}

type BigIDExample struct {
	ID  core.BigID   `json:"id"`
	Arr []core.BigID `json:"arr"`
	Age int          `json:"age"`
}

func TestBigIDMarshalJsonExp(t *testing.T) {
	exp := BigIDExample{
		ID: core.NewBigID(),
		Arr: []core.BigID{
			core.NewBigID(),
			core.NewBigID(),
			core.NewBigID(),
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
