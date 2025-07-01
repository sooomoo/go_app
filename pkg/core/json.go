package core

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

type SqlJSON map[string]any

var EmptySqlJSON = SqlJSON{}

func (j SqlJSON) GetString(key string) string {
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

// Value return json value, implement driver.Valuer interface
func (j SqlJSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	jsonStr, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(jsonStr), nil
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (j *SqlJSON) Scan(value any) error {
	if value == nil {
		*j = SqlJSON{}
		return nil
	}
	var bytes []byte
	if s, ok := value.(fmt.Stringer); ok {
		bytes = []byte(s.String())
	} else {
		switch v := value.(type) {
		case []byte:
			if len(v) > 0 {
				bytes = make([]byte, len(v))
				copy(bytes, v)
			}
		case string:
			bytes = []byte(v)
		default:
			return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
		}
	}

	var mp SqlJSON
	err := json.Unmarshal(bytes, &mp)
	*j = mp
	return err
}

func (j SqlJSON) String() string {
	if len(j) == 0 {
		return ""
	}
	data, err := json.Marshal(j)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

// GormDataType gorm common data type
func (SqlJSON) GormDataType() string {
	return "json"
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
	}
	return ""
}

func (js SqlJSON) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	if len(js) == 0 {
		return gorm.Expr("NULL")
	}

	data, _ := json.Marshal(js)

	switch db.Dialector.Name() {
	case "mysql":
		if v, ok := db.Dialector.(*mysql.Dialector); ok && !strings.Contains(v.ServerVersion, "MariaDB") {
			return gorm.Expr("CAST(? AS JSON)", string(data))
		}
	}

	return gorm.Expr("?", string(data))
}
