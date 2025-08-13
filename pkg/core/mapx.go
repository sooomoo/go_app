package core

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

type MapX map[string]any

var _ json.Marshaler = (*MapX)(nil)
var _ json.Unmarshaler = (*MapX)(nil)

// MarshalJSON to output non base64 encoded []byte
func (m MapX) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	t := (map[string]any)(m)
	return json.Marshal(t)
}

// UnmarshalJSON to deserialize []byte
func (m *MapX) UnmarshalJSON(b []byte) error {
	t := map[string]any{}
	err := json.Unmarshal(b, &t)
	*m = MapX(t)
	return err
}

// 按路径读取值:支持动态深度
func (e MapX) GetValue(path string) (any, bool) {
	if e.IsEmpty() {
		return nil, false
	}
	keys := strings.Split(path, ".")  // 分割路径为键序列
	current := any(map[string]any(e)) // 初始化为顶层 map

	for _, key := range keys {
		if idx, err := strconv.Atoi(key); err == nil {
			// 是数组下标
			// 验证当前层是否为切片
			slice, ok := current.([]any)
			if !ok {
				return nil, false
			}
			// 检查索引范围
			if idx < 0 || idx >= len(slice) {
				return nil, false
			}
			current = slice[idx] // 定位到数组元素
		} else {
			// 1. 将当前层转换为 map[string]any
			currentMap, ok := current.(map[string]any)
			if !ok {
				return nil, false // 当前层不是 map
			}

			// 2. 检查键是否存在
			val, exists := currentMap[key]
			if !exists {
				return nil, false // 键不存在
			}

			// 3. 更新当前层为下一级对象
			current = val
		}
	}
	return current, true // 返回最终值
}

func (e MapX) SetValue(path string, value any) error {
	if e == nil {
		return errors.New("mapx is nil")
	}
	keys := strings.Split(path, ".") // 分割路径为键序列
	parent := any(map[string]any(e))
	for i := 0; i < len(keys)-1; i++ {
		curKey := keys[i]
		nextKey := keys[i+1]
		curValueType := "map"
		if _, err := strconv.Atoi(nextKey); err == nil {
			// 下一个 key 是数组下标, 说明需要定位到数组
			curValueType = "slice"
		}

		if pmap, ok := parent.(map[string]any); ok {
			if _, ok := pmap[curKey]; !ok {
				if curValueType == "slice" {
					pmap[curKey] = []any{}
				} else {
					pmap[curKey] = make(map[string]any)
				}
			}
			parent = pmap[curKey]
		} else if parr, ok := parent.([]any); ok {
			// 此时 curKey 肯定为数组下标
			parrIdx, err := strconv.Atoi(curKey)
			if err != nil {
				return err
			}
			if parrIdx < 0 || parrIdx >= len(parr) {
				return errors.New("out of range")
			}

			if parr[parrIdx] == nil {
				if curValueType == "slice" {
					parr[parrIdx] = []any{}
				} else {
					parr[parrIdx] = make(map[string]any)
				}
			}

			parent = parr[parrIdx]
		}
	}

	// 赋值到叶子节点
	leafKey := keys[len(keys)-1]
	if arr, ok := parent.([]any); ok {
		arrIdx, err := strconv.Atoi(leafKey)
		if err != nil {
			return err
		}
		if arrIdx < 0 || arrIdx > len(arr)-1 {
			return errors.New("out of index")
		}
		arr[arrIdx] = value
	} else if mapArr, ok := parent.(map[string]any); ok {
		mapArr[leafKey] = value
	}
	return nil
}

func (e MapX) Delete(key string) {
	delete(e, key)
}

func (e MapX) GetString(path string, def string) string {
	v, ok := e.GetValue(path)
	if !ok {
		return def
	}
	if s, ok := v.(string); ok {
		return s
	}
	return def
}

func (e MapX) GetBool(path string, def bool) bool {
	val, ok := e.GetValue(path)
	if !ok {
		return def
	}

	if i, ok := val.(bool); ok {
		return i
	}
	defVal := 0
	if def {
		defVal = 1
	}
	intVal := e.GetInt(path, defVal)
	return intVal > 0
}

func (e MapX) GetInt(path string, def int) int {
	val, ok := e.GetValue(path)
	if !ok {
		return def
	}

	if i, ok := val.(int); ok {
		return i
	}
	if i, ok := val.(int64); ok {
		return int(i)
	}
	if i, ok := val.(int32); ok {
		return int(i)
	}
	if i, ok := val.(int16); ok {
		return int(i)
	}
	if i, ok := val.(int8); ok {
		return int(i)
	}
	return 0
}

func (e MapX) GetInt64(path string, def int64) int64 {
	return int64(e.GetInt(path, int(def)))
}

func (e MapX) GetInt32(path string, def int32) int32 {
	return int32(e.GetInt(path, int(def)))
}

func (e MapX) Clear() {
	clear(e)
}

func (e MapX) Len() int {
	return len(e)
}

func (e MapX) IsEmpty() bool {
	return len(e) == 0
}
