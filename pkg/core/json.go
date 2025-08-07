package core

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"goapp/pkg/strs"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type SqlJSON map[string]any

var _ json.Marshaler = (*SqlJSON)(nil)
var _ json.Unmarshaler = (*SqlJSON)(nil)

var EmptySqlJSON = SqlJSON{}

func GetSqlJSONValue[T any](sqljson SqlJSON, path string) T {
	v := sqljson.Get(path)
	if vv, ok := v.(T); ok {
		return vv
	}
	var t T
	return t
}

func (j SqlJSON) GetValue(key string) any {
	if len(j) == 0 {
		return nil
	}
	if v, ok := j[key]; ok {
		return v
	}
	return nil
}

// 支持按路径获取值, 例如: a.b.c
func (j SqlJSON) Get(path string) any {
	if len(j) == 0 {
		return nil
	}
	iterFn := func(key string, target map[string]any) any {
		if v, ok := target[key]; ok {
			return v
		}
		return nil
	}
	pathSpl := strs.Split(path, ".")
	if len(pathSpl) == 1 {
		return iterFn(pathSpl[0], j)
	}

	currentTarget := (map[string]any)(j)
	count := len(pathSpl)
	for i, seg := range pathSpl {
		val := iterFn(seg, currentTarget)
		if val == nil || i == count-1 {
			return val
		}
		if t, ok := val.(map[string]any); ok {
			currentTarget = t
			continue
		}
		return val
	}
	return nil
}

func (j SqlJSON) GetBool(path string) bool {
	v := j.Get(path)
	if vv, ok := v.(bool); ok {
		return vv
	}
	val := j.GetInt64(path)
	return val > 0
}

func (j SqlJSON) GetString(path string) string {
	v := j.Get(path)
	if vv, ok := v.(string); ok {
		return vv
	}
	return ""
}

func (j SqlJSON) GetInt64(path string) int64 {
	v := j.Get(path)
	if vv, ok := v.(int64); ok {
		return vv
	}
	if vv, ok := v.(int); ok {
		return int64(vv)
	}
	if vv, ok := v.(int32); ok {
		return int64(vv)
	}
	if vv, ok := v.(int8); ok {
		return int64(vv)
	}
	return 0
}

func (j SqlJSON) GetInt(path string) int {
	v := j.Get(path)
	if vv, ok := v.(int); ok {
		return vv
	}
	if vv, ok := v.(int32); ok {
		return int(vv)
	}
	if vv, ok := v.(int64); ok {
		return int(vv)
	}
	if vv, ok := v.(int8); ok {
		return int(vv)
	}
	return 0
}

func (j SqlJSON) GetInt32(path string) int32 {
	v := j.Get(path)
	if vv, ok := v.(int); ok {
		return int32(vv)
	}
	if vv, ok := v.(int32); ok {
		return vv
	}
	if vv, ok := v.(int64); ok {
		return int32(vv)
	}
	if vv, ok := v.(int8); ok {
		return int32(vv)
	}
	return 0
}

// Value return json value, implement driver.Valuer interface
func (m SqlJSON) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	ba, err := m.MarshalJSON()
	return string(ba), err
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (m *SqlJSON) Scan(val any) error {
	if val == nil {
		*m = make(SqlJSON)
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", val))
	}
	t := map[string]any{}
	rd := bytes.NewReader(ba)
	decoder := json.NewDecoder(rd)
	decoder.UseNumber()
	err := decoder.Decode(&t)
	*m = t
	return err
}

// MarshalJSON to output non base64 encoded []byte
func (m SqlJSON) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	t := (map[string]any)(m)
	return json.Marshal(t)
}

// UnmarshalJSON to deserialize []byte
func (m *SqlJSON) UnmarshalJSON(b []byte) error {
	t := map[string]any{}
	err := json.Unmarshal(b, &t)
	*m = SqlJSON(t)
	return err
}

// GormDataType gorm common data type
func (m SqlJSON) GormDataType() string {
	return "jsonmap"
}

// GormDBDataType gorm db data type
func (SqlJSON) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	case "mysql":
		return "JSON"
	case "postgres":
		return "JSONB"
	case "sqlserver":
		return "NVARCHAR(MAX)"
	}
	return ""
}

func (jm SqlJSON) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	data, _ := jm.MarshalJSON()
	switch db.Dialector.Name() {
	case "mysql":
		if v, ok := db.Dialector.(*mysql.Dialector); ok && !strings.Contains(v.ServerVersion, "MariaDB") {
			return gorm.Expr("CAST(? AS JSON)", string(data))
		}
	}
	return gorm.Expr("?", string(data))
}
