package db

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"goapp/pkg/core"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// 用于表示数据库中 json、jsonb 类型
type JSON core.MapX

var _ json.Marshaler = (*JSON)(nil)
var _ json.Unmarshaler = (*JSON)(nil)

func (j JSON) GetValue(path string) (any, bool) {
	return core.MapX(j).GetValue(path)
}

func (j JSON) GetBool(path string, def bool) bool {
	return core.MapX(j).GetBool(path, def)
}

func (j JSON) GetString(path string, def string) string {
	return core.MapX(j).GetString(path, def)
}

func (j JSON) GetInt64(path string, def int64) int64 {
	return core.MapX(j).GetInt64(path, def)
}

func (j JSON) GetInt(path string, def int) int {
	return core.MapX(j).GetInt(path, def)
}

func (j JSON) GetInt32(path string, def int32) int32 {
	return core.MapX(j).GetInt32(path, def)
}

func (j JSON) SetValue(path string, value any) error {
	return core.MapX(j).SetValue(path, value)
}

func (j JSON) Delete(key string) {
	delete(j, key)
}

func (j JSON) Clear() {
	clear(j)
}

func (j JSON) Len() int {
	return len(j)
}

func (j JSON) IsEmpty() bool {
	return len(j) == 0
}

// Value return json value, implement driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	ba, err := j.MarshalJSON()
	return ba, err
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (j *JSON) Scan(val any) error {
	if val == nil {
		*j = make(JSON)
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
	*j = t
	return err
}

// MarshalJSON to output non base64 encoded []byte
func (j JSON) MarshalJSON() ([]byte, error) {
	return core.MapX(j).MarshalJSON()
}

// UnmarshalJSON to deserialize []byte
func (j *JSON) UnmarshalJSON(b []byte) error {
	// t := map[string]any{}
	// err := json.Unmarshal(b, &t)
	// *j = JSON(t)
	return (*core.MapX)(j).UnmarshalJSON(b)
}

// GormDataType gorm common data type
func (j JSON) GormDataType() string {
	return "db.JSON"
}

// GormDBDataType gorm db data type
func (JSON) GormDBDataType(db *gorm.DB, field *schema.Field) string {
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

func (jm JSON) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	data, _ := jm.MarshalJSON()
	switch db.Dialector.Name() {
	case "mysql":
		if v, ok := db.Dialector.(*mysql.Dialector); ok && !strings.Contains(v.ServerVersion, "MariaDB") {
			return gorm.Expr("CAST(? AS JSON)", string(data))
		}
	case "postgres":
		return gorm.Expr("?", data)
	}
	return gorm.Expr("?", string(data))
}
