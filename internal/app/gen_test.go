package app_test

import (
	"fmt"
	"goapp/pkg/ids"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func underScoreToCamelCase(name string) string {
	segs := strings.Split(name, "_")
	trimedSegs := []string{}
	for _, seg := range segs {
		trim := strings.TrimSpace(seg)
		if len(trim) == 0 {
			continue
		}
		trimedSegs = append(trimedSegs, trim)
	}
	builder := strings.Builder{}
	for i, v := range trimedSegs {
		if i == 0 {
			builder.WriteString(strings.ToLower(v))
		} else {
			camelCase := strings.ToUpper(v[:1]) + v[1:]
			builder.WriteString(camelCase)
		}
	}
	return builder.String()
}

// 可以通过以下方式，为 binary 的 id 创建虚拟列，以便于查看
// ALTER TABLE users
// ADD COLUMN id_hex VARCHAR(24) GENERATED ALWAYS AS (
//
//	HEX(id)
//
// ) VIRTUAL AFTER `id`;
func TestGenDao(t *testing.T) {
	g := gen.NewGenerator(gen.Config{
		OutPath:      "./dao/query",
		ModelPkgPath: "./dao/model",

		Mode: gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface, // generate mode
	})

	gormdb, _ := gorm.Open(mysql.Open("root:abc12345@tcp(localhost:3306)/niu?charset=utf8mb4&parseTime=true&loc=Local"))
	g.UseDB(gormdb) // reuse your gorm db
	g.WithJSONTagNameStrategy(underScoreToCamelCase)
	g.WithDataTypeMap(map[string]func(columnType gorm.ColumnType) (dataType string){
		// "binary": func(columnType gorm.ColumnType) (dataType string) {
		// 	name := strings.ToLower(columnType.Name())
		// 	if name == "id" || strings.HasSuffix(name, "_id") {
		// 		dataType = "core.SeqID"
		// 	} else {
		// 		dataType = "[]byte"
		// 	}

		// 	fmt.Println(name)
		// 	return
		// },
		"json": func(columnType gorm.ColumnType) (dataType string) {
			name := strings.ToLower(columnType.Name())
			dataType = "db.JSON"
			fmt.Println(name)
			return
		},
		"tinyint": func(columnType gorm.ColumnType) (dataType string) {
			name := strings.ToLower(columnType.Name())
			dataType = "uint8"
			fmt.Println(name)
			return
		},
	})

	// Generate basic type-safe DAO API for struct `model.User` following conventions
	g.ApplyBasic(
		// Generate structs from all tables of current database
		g.GenerateAllTable()...,
	)
	// Generate the code
	g.Execute()
}

func TestGenDaoPostgreSQL(t *testing.T) {
	g := gen.NewGenerator(gen.Config{
		OutPath:      "./dao/query",
		ModelPkgPath: "./dao/model",

		Mode: gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface, // generate mode
	})
	dsn := "host=localhost user=postgres password=abc12345 " +
		"dbname=dev_db port=5432 sslmode=disable TimeZone=Asia/Shanghai "
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // 输出到控制台
		logger.Config{
			SlowThreshold:             time.Millisecond * 200, // 慢查询阈值（超过200毫秒标记）
			LogLevel:                  logger.Info,            // 输出所有SQL（包括参数、耗时）
			Colorful:                  true,                   // 彩色输出
			IgnoreRecordNotFoundError: true,                   // 忽略"未找到记录"错误
		},
	)
	// 建立连接
	gormdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		panic("连接失败: " + err.Error())
	}
	g.UseDB(gormdb) // reuse your gorm db
	g.WithJSONTagNameStrategy(underScoreToCamelCase)
	g.WithDataTypeMap(map[string]func(columnType gorm.ColumnType) (dataType string){
		"json": func(columnType gorm.ColumnType) (dataType string) {
			name := strings.ToLower(columnType.Name())
			dataType = "db.JSON"
			fmt.Println(name)
			return
		},
		"jsonb": func(columnType gorm.ColumnType) (dataType string) {
			name := strings.ToLower(columnType.Name())
			dataType = "db.JSON"
			fmt.Println(name)
			return
		},
		"uuid": func(columnType gorm.ColumnType) (dataType string) {
			name := strings.ToLower(columnType.Name())
			fmt.Println(name)
			dataType = "ids.UID"
			return
		},
	})

	// Generate basic type-safe DAO API for struct `model.User` following conventions
	g.ApplyBasic(
		// Generate structs from all tables of current database
		g.GenerateAllTable()...,
	)
	// Generate the code
	g.Execute()

	dev := Devable{
		ID:   ids.NewUID(),
		Name: "abc " + strconv.FormatInt(time.Now().Unix(), 10),
	}
	gormdb.Model(&dev).Create(&dev)
	var dev2 []Devable
	gormdb.Table("devable").Find(&dev2)
	fmt.Println(dev2)
}

const TableNameDevable = "devable"

// Devable mapped from table <devable>
type Devable struct {
	ID   ids.UID `gorm:"column:id;primaryKey" json:"id"`
	Name string  `gorm:"column:name;not null" json:"name"`
}

// TableName Devable's table name
func (*Devable) TableName() string {
	return TableNameDevable
}
