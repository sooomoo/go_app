package core

import "encoding/json"

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

func (e MapX) Get(key string) any {
	if v, ok := e[key]; ok {
		return v
	}
	return nil
}

func (e MapX) GetString(key string) string {
	if v, ok := e[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (e MapX) GetInt(key string) int {
	if v, ok := e[key]; ok {
		if i, ok := v.(int); ok {
			return i
		}
		if i, ok := v.(int64); ok {
			return int(i)
		}
		if i, ok := v.(int32); ok {
			return int(i)
		}
		if i, ok := v.(int16); ok {
			return int(i)
		}
		if i, ok := v.(int8); ok {
			return int(i)
		}
	}
	return 0
}

func (e MapX) GetInt64(key string) int64 {
	if v, ok := e[key]; ok {
		if i, ok := v.(int64); ok {
			return i
		}
		if i, ok := v.(int); ok {
			return int64(i)
		}
	}
	return 0
}

func (e MapX) GetInt32(key string) int32 {
	if v, ok := e[key]; ok {
		if i, ok := v.(int32); ok {
			return i
		}
		if i, ok := v.(int); ok {
			return int32(i)
		}
	}
	return 0
}

func (e MapX) Set(key string, value any) {
	e[key] = value
}
func (e MapX) Delete(key string) {
	delete(e, key)
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
