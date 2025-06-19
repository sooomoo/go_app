package pkg_test

import (
	"encoding/json"
	"fmt"
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
