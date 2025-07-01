package core

import (
	"encoding/json"

	"gorm.io/datatypes"
)

type JSONMap map[string]any

func (j JSONMap) GetString(key string) string {
	if len(j) == 0 {
		return ""
	}
	if v, ok := j[key]; ok {
		if vv, ok := v.(string); ok {
			return vv
		}
	}
	return ""
}

// Json 序列化
func JsonMarshal(v any) string {
	jsonStr, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(jsonStr)
}

func JsonUnmarshalSqlJSON(jsonStr datatypes.JSON) JSONMap {
	var mp JSONMap
	json.Unmarshal([]byte(jsonStr), &mp)
	return mp
}
