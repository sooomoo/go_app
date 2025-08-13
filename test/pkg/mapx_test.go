package pkg

import (
	"encoding/json"
	"fmt"
	"goapp/pkg/core"
	"testing"
)

func TestMapXGetValue(t *testing.T) {
	// 测试数据（支持 map + 嵌套切片）
	data := core.MapX{
		"a": map[string]any{
			"b": []any{
				map[string]any{"c": "值1"},
				map[string]any{"c": "值2"},
			},
		},
		"ar": []any{100, 200},
		"x": []any{
			"直接字符串",
			[]any{"嵌套数组元素", 100},
		},
	}

	// 测试路径
	paths := []string{
		"a.b.0.c",     // ✅ 正常访问
		"ar.1",        // ✅ 正常访问
		"x.0",         // ✅ 访问切片元素
		"a.b.1.c",     // ✅ 嵌套第二层
		"a.b.2",       // ❌ 索引越界（切片长度=2）
		"invalid.key", // ❌ 键不存在
		"x.1.0",       // ✅ 嵌套切片中的字符串
	}

	err := data.SetValue("a.b.0.c", "值1 updated")
	if err != nil {
		fmt.Println("err", err)
	}
	err = data.SetValue("c.d", 300)
	if err != nil {
		fmt.Println("err", err)
	}
	err = data.SetValue("e.f.g", "sdfafd00")
	if err != nil {
		fmt.Println("err", err)
	}

	for _, path := range paths {
		val, ok := data.GetValue(path)
		if ok {
			fmt.Printf("✅ [%s] = %v (Type: %T)\n", path, val, val)
		} else {
			fmt.Printf("❌ [%s]\n", path)
		}
	}

	out, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(out))
}
