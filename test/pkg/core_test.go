package pkg_test

import (
	"encoding/json"
	"fmt"
	"goapp/pkg/collection"
	"goapp/pkg/core"
	"sync"
	"testing"
)

type Example struct {
	ID    int64  `json:"id,string"` // 增加 string 标签，序列化时转为字符串
	Name  string `json:"name"`
	Value int64  `json:"value"` // 保持为数字
	Ptr   *int64 `json:"ptr"`
}

func TestInt64String(t *testing.T) {
	data := Example{
		ID:    1234567890123,
		Name:  "test",
		Value: 987654321,
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

	set := collection.Set[string]{}
	cnt := 2000
	wg := sync.WaitGroup{}
	wg.Add(cnt)
	for range cnt {
		go func() {
			defer wg.Done()
			for range 10000 {
				id := core.NewSeqID().Hex()
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

type SeqIDExample struct {
	ID  core.SeqID `json:"id"`
	Age int        `json:"age"`
}

func TestSeqIDMarshalJsonExp(t *testing.T) {
	exp := SeqIDExample{
		ID:  core.NewSeqID(),
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
