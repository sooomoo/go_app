package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// List 用于将任意结构体数组序列化为 json 存储到数据库中
//
// T 可以是任意类型
type List[T any] []T

// implements sql.Scanner interface
func (j *List[T]) Scan(value any) error {
	if value == nil {
		return nil
	}
	var rawBytes []byte
	switch value := value.(type) {
	case []byte:
		rawBytes = value
	case string:
		rawBytes = []byte(value)
	}
	if len(rawBytes) == 0 || string(rawBytes) == "null" {
		return nil
	}

	var temp []T
	if err := json.Unmarshal(rawBytes, &temp); err != nil {
		return err
	}
	*j = temp
	return nil
}

// implement driver.Valuer interface
func (j List[T]) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(j)
	if err != nil {
		fmt.Println(err)
	}
	return data, err
}

// 实现 GormDBDataTypeInterface
func (List[T]) GormDBDataType(db *gorm.DB, field *schema.Field) string {
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

// ​GormValuerInterface接口​：这是 GORM 提供的扩展接口，要求实现
//
//	GormValue(ctx context.Context, db *gorm.DB) (clause.Expr)
//
// 方法，它的能力更强，允许你返回一个完整的 SQL 表达式（clause.Expr），而不仅仅是一个简单的值
// 这使得你可以嵌入数据库函数或构建更复杂的逻辑。
//
//   - 如果没有复杂的需求，不实现此接口也可以
// func (jm DBList[T]) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
// 	if len(jm) == 0 {
// 		// 返回一个表示 SQL NULL 的表达式
// 		return clause.Expr{SQL: "NULL"}
// 	}
// 	data, _ := json.Marshal(jm)
// 	switch db.Dialector.Name() {
// 	case "mysql":
// 		if v, ok := db.Dialector.(*mysql.Dialector); ok && !strings.Contains(v.ServerVersion, "MariaDB") {
// 			return gorm.Expr("CAST(? AS JSON)", string(data))
// 		}
// 	case "postgres":
// 		return gorm.Expr("?", data)
// 	}
// 	return gorm.Expr("?", string(data))
// }

// Object 用于将任意结构体序列化为 json 存储到数据库中
//
// Target 是指向 T 类型的指针; T 必须不是指针类型，因为 Target 已经是指针了
//
// 例如： Object[MyStruct]，而不是 Object[*MyStruct]
//
// 在指定 Object 作为结构体字段时，不能使用指针类型，例如：
//
//	type MyEntity struct {
//	    Data entities.Object[MyStruct] // 正确
//	    DataPtr *entities.Object[MyStruct] // 错误，不能使用指针类型
//	}
//
// 如果不按照要求使用，可能出现意想不到的错误
type Object[T any] struct {
	Target *T
}

// implements sql.Scanner interface
func (j *Object[T]) Scan(value any) error {
	if value == nil {
		*j = Object[T]{}
		return nil
	}
	var rawBytes []byte
	switch value := value.(type) {
	case []byte:
		rawBytes = value
	case string:
		rawBytes = []byte(value)
	}
	if len(rawBytes) == 0 || string(rawBytes) == "null" {
		*j = Object[T]{}
		return nil
	}

	var temp T
	if err := json.Unmarshal(rawBytes, &temp); err != nil {
		*j = Object[T]{}
		return err
	}
	*j = Object[T]{Target: &temp}
	return nil
}

// implement driver.Valuer interface
func (j Object[T]) Value() (driver.Value, error) {
	if j.Target == nil {
		return nil, nil
	}
	return json.Marshal(j.Target)
}

func (id Object[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.Target)
}

func (obj *Object[T]) UnmarshalJSON(b []byte) error {
	// Ignore "null" to keep parity with the standard library. Decoding a JSON
	// null into a non-pointer SeqID field will leave the field unchanged.
	// For pointer values, encoding/json will set the pointer to nil and will
	// not enter the UnmarshalJSON hook.
	if string(b) == "null" || string(b) == "NULL" {
		*obj = Object[T]{Target: nil}
		return nil
	}

	var err error
	var target T
	// Handle string
	if len(b) >= 2 && b[0] == '"' {
		err = json.Unmarshal(b[1:len(b)-1], &target)
	} else {
		err = json.Unmarshal(b, &target)
	}

	*obj = Object[T]{Target: &target}
	return err
}

// 实现 GormDBDataTypeInterface
func (Object[T]) GormDBDataType(db *gorm.DB, field *schema.Field) string {
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

// ​GormValuerInterface接口​：这是 GORM 提供的扩展接口，要求实现
//
//	GormValue(ctx context.Context, db *gorm.DB) (clause.Expr)
//
// 方法，它的能力更强，允许你返回一个完整的 SQL 表达式（clause.Expr），而不仅仅是一个简单的值
// 这使得你可以嵌入数据库函数或构建更复杂的逻辑。
//
//   - 如果没有复杂的需求，不实现此接口也可以
// func (jm DBObject[T]) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
// 	if jm.Target == nil {
// 		// 返回一个表示 SQL NULL 的表达式
// 		return clause.Expr{SQL: "NULL"}
// 	}
// 	data, _ := jm.MarshalJSON()
// 	switch db.Dialector.Name() {
// 	case "mysql":
// 		if v, ok := db.Dialector.(*mysql.Dialector); ok && !strings.Contains(v.ServerVersion, "MariaDB") {
// 			return gorm.Expr("CAST(? AS JSON)", string(data))
// 		}
// 	case "postgres":
// 		return gorm.Expr("?", data)
// 	}
// 	return gorm.Expr("?", string(data))
// }
