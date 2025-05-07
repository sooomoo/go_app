package repository_test

import (
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
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

func TestGenDao(t *testing.T) {
	g := gen.NewGenerator(gen.Config{
		OutPath:      "./dao/query",
		ModelPkgPath: "./dao/model",

		Mode: gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface, // generate mode
	})

	gormdb, _ := gorm.Open(mysql.Open("root:abc12345@tcp(localhost:3306)/niu?charset=utf8mb4&parseTime=true&loc=Local"))
	g.UseDB(gormdb) // reuse your gorm db
	g.WithJSONTagNameStrategy(underScoreToCamelCase)

	// Generate basic type-safe DAO API for struct `model.User` following conventions
	g.ApplyBasic(
		// Generate structs from all tables of current database
		g.GenerateAllTable()...,
	)
	// Generate the code
	g.Execute()
}
