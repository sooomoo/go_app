// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package query

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"gorm.io/gen"
	"gorm.io/gen/field"

	"gorm.io/plugin/dbresolver"

	"goapp/internal/app/stores/dao/model"
)

func newTaskWebSearch(db *gorm.DB, opts ...gen.DOOption) taskWebSearch {
	_taskWebSearch := taskWebSearch{}

	_taskWebSearch.taskWebSearchDo.UseDB(db, opts...)
	_taskWebSearch.taskWebSearchDo.UseModel(&model.TaskWebSearch{})

	tableName := _taskWebSearch.taskWebSearchDo.TableName()
	_taskWebSearch.ALL = field.NewAsterisk(tableName)
	_taskWebSearch.ID = field.NewInt64(tableName, "id")
	_taskWebSearch.Keywords = field.NewString(tableName, "keywords")
	_taskWebSearch.TraceID = field.NewString(tableName, "trace_id")
	_taskWebSearch.Status = field.NewUint8(tableName, "status")
	_taskWebSearch.StatusText = field.NewString(tableName, "status_text")
	_taskWebSearch.Result = field.NewField(tableName, "result")
	_taskWebSearch.CreatedAt = field.NewInt64(tableName, "created_at")
	_taskWebSearch.SearchAt = field.NewInt64(tableName, "search_at")
	_taskWebSearch.FinishAt = field.NewInt64(tableName, "finish_at")

	_taskWebSearch.fillFieldMap()

	return _taskWebSearch
}

// taskWebSearch 搜索任务
type taskWebSearch struct {
	taskWebSearchDo

	ALL        field.Asterisk
	ID         field.Int64
	Keywords   field.String // 搜索词，多个搜索词以逗号分隔
	TraceID    field.String // 多个 ID 以逗号分隔
	Status     field.Uint8
	StatusText field.String
	Result     field.Field
	CreatedAt  field.Int64 // 由 app 创建
	SearchAt   field.Int64 // 什么时候开始的搜索：由 searcher 更新
	FinishAt   field.Int64 // 什么时候结束的搜索：由 searcher 更新

	fieldMap map[string]field.Expr
}

func (t taskWebSearch) Table(newTableName string) *taskWebSearch {
	t.taskWebSearchDo.UseTable(newTableName)
	return t.updateTableName(newTableName)
}

func (t taskWebSearch) As(alias string) *taskWebSearch {
	t.taskWebSearchDo.DO = *(t.taskWebSearchDo.As(alias).(*gen.DO))
	return t.updateTableName(alias)
}

func (t *taskWebSearch) updateTableName(table string) *taskWebSearch {
	t.ALL = field.NewAsterisk(table)
	t.ID = field.NewInt64(table, "id")
	t.Keywords = field.NewString(table, "keywords")
	t.TraceID = field.NewString(table, "trace_id")
	t.Status = field.NewUint8(table, "status")
	t.StatusText = field.NewString(table, "status_text")
	t.Result = field.NewField(table, "result")
	t.CreatedAt = field.NewInt64(table, "created_at")
	t.SearchAt = field.NewInt64(table, "search_at")
	t.FinishAt = field.NewInt64(table, "finish_at")

	t.fillFieldMap()

	return t
}

func (t *taskWebSearch) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := t.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (t *taskWebSearch) fillFieldMap() {
	t.fieldMap = make(map[string]field.Expr, 9)
	t.fieldMap["id"] = t.ID
	t.fieldMap["keywords"] = t.Keywords
	t.fieldMap["trace_id"] = t.TraceID
	t.fieldMap["status"] = t.Status
	t.fieldMap["status_text"] = t.StatusText
	t.fieldMap["result"] = t.Result
	t.fieldMap["created_at"] = t.CreatedAt
	t.fieldMap["search_at"] = t.SearchAt
	t.fieldMap["finish_at"] = t.FinishAt
}

func (t taskWebSearch) clone(db *gorm.DB) taskWebSearch {
	t.taskWebSearchDo.ReplaceConnPool(db.Statement.ConnPool)
	return t
}

func (t taskWebSearch) replaceDB(db *gorm.DB) taskWebSearch {
	t.taskWebSearchDo.ReplaceDB(db)
	return t
}

type taskWebSearchDo struct{ gen.DO }

type ITaskWebSearchDo interface {
	gen.SubQuery
	Debug() ITaskWebSearchDo
	WithContext(ctx context.Context) ITaskWebSearchDo
	WithResult(fc func(tx gen.Dao)) gen.ResultInfo
	ReplaceDB(db *gorm.DB)
	ReadDB() ITaskWebSearchDo
	WriteDB() ITaskWebSearchDo
	As(alias string) gen.Dao
	Session(config *gorm.Session) ITaskWebSearchDo
	Columns(cols ...field.Expr) gen.Columns
	Clauses(conds ...clause.Expression) ITaskWebSearchDo
	Not(conds ...gen.Condition) ITaskWebSearchDo
	Or(conds ...gen.Condition) ITaskWebSearchDo
	Select(conds ...field.Expr) ITaskWebSearchDo
	Where(conds ...gen.Condition) ITaskWebSearchDo
	Order(conds ...field.Expr) ITaskWebSearchDo
	Distinct(cols ...field.Expr) ITaskWebSearchDo
	Omit(cols ...field.Expr) ITaskWebSearchDo
	Join(table schema.Tabler, on ...field.Expr) ITaskWebSearchDo
	LeftJoin(table schema.Tabler, on ...field.Expr) ITaskWebSearchDo
	RightJoin(table schema.Tabler, on ...field.Expr) ITaskWebSearchDo
	Group(cols ...field.Expr) ITaskWebSearchDo
	Having(conds ...gen.Condition) ITaskWebSearchDo
	Limit(limit int) ITaskWebSearchDo
	Offset(offset int) ITaskWebSearchDo
	Count() (count int64, err error)
	Scopes(funcs ...func(gen.Dao) gen.Dao) ITaskWebSearchDo
	Unscoped() ITaskWebSearchDo
	Create(values ...*model.TaskWebSearch) error
	CreateInBatches(values []*model.TaskWebSearch, batchSize int) error
	Save(values ...*model.TaskWebSearch) error
	First() (*model.TaskWebSearch, error)
	Take() (*model.TaskWebSearch, error)
	Last() (*model.TaskWebSearch, error)
	Find() ([]*model.TaskWebSearch, error)
	FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.TaskWebSearch, err error)
	FindInBatches(result *[]*model.TaskWebSearch, batchSize int, fc func(tx gen.Dao, batch int) error) error
	Pluck(column field.Expr, dest interface{}) error
	Delete(...*model.TaskWebSearch) (info gen.ResultInfo, err error)
	Update(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	Updates(value interface{}) (info gen.ResultInfo, err error)
	UpdateColumn(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateColumnSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	UpdateColumns(value interface{}) (info gen.ResultInfo, err error)
	UpdateFrom(q gen.SubQuery) gen.Dao
	Attrs(attrs ...field.AssignExpr) ITaskWebSearchDo
	Assign(attrs ...field.AssignExpr) ITaskWebSearchDo
	Joins(fields ...field.RelationField) ITaskWebSearchDo
	Preload(fields ...field.RelationField) ITaskWebSearchDo
	FirstOrInit() (*model.TaskWebSearch, error)
	FirstOrCreate() (*model.TaskWebSearch, error)
	FindByPage(offset int, limit int) (result []*model.TaskWebSearch, count int64, err error)
	ScanByPage(result interface{}, offset int, limit int) (count int64, err error)
	Rows() (*sql.Rows, error)
	Row() *sql.Row
	Scan(result interface{}) (err error)
	Returning(value interface{}, columns ...string) ITaskWebSearchDo
	UnderlyingDB() *gorm.DB
	schema.Tabler
}

func (t taskWebSearchDo) Debug() ITaskWebSearchDo {
	return t.withDO(t.DO.Debug())
}

func (t taskWebSearchDo) WithContext(ctx context.Context) ITaskWebSearchDo {
	return t.withDO(t.DO.WithContext(ctx))
}

func (t taskWebSearchDo) ReadDB() ITaskWebSearchDo {
	return t.Clauses(dbresolver.Read)
}

func (t taskWebSearchDo) WriteDB() ITaskWebSearchDo {
	return t.Clauses(dbresolver.Write)
}

func (t taskWebSearchDo) Session(config *gorm.Session) ITaskWebSearchDo {
	return t.withDO(t.DO.Session(config))
}

func (t taskWebSearchDo) Clauses(conds ...clause.Expression) ITaskWebSearchDo {
	return t.withDO(t.DO.Clauses(conds...))
}

func (t taskWebSearchDo) Returning(value interface{}, columns ...string) ITaskWebSearchDo {
	return t.withDO(t.DO.Returning(value, columns...))
}

func (t taskWebSearchDo) Not(conds ...gen.Condition) ITaskWebSearchDo {
	return t.withDO(t.DO.Not(conds...))
}

func (t taskWebSearchDo) Or(conds ...gen.Condition) ITaskWebSearchDo {
	return t.withDO(t.DO.Or(conds...))
}

func (t taskWebSearchDo) Select(conds ...field.Expr) ITaskWebSearchDo {
	return t.withDO(t.DO.Select(conds...))
}

func (t taskWebSearchDo) Where(conds ...gen.Condition) ITaskWebSearchDo {
	return t.withDO(t.DO.Where(conds...))
}

func (t taskWebSearchDo) Order(conds ...field.Expr) ITaskWebSearchDo {
	return t.withDO(t.DO.Order(conds...))
}

func (t taskWebSearchDo) Distinct(cols ...field.Expr) ITaskWebSearchDo {
	return t.withDO(t.DO.Distinct(cols...))
}

func (t taskWebSearchDo) Omit(cols ...field.Expr) ITaskWebSearchDo {
	return t.withDO(t.DO.Omit(cols...))
}

func (t taskWebSearchDo) Join(table schema.Tabler, on ...field.Expr) ITaskWebSearchDo {
	return t.withDO(t.DO.Join(table, on...))
}

func (t taskWebSearchDo) LeftJoin(table schema.Tabler, on ...field.Expr) ITaskWebSearchDo {
	return t.withDO(t.DO.LeftJoin(table, on...))
}

func (t taskWebSearchDo) RightJoin(table schema.Tabler, on ...field.Expr) ITaskWebSearchDo {
	return t.withDO(t.DO.RightJoin(table, on...))
}

func (t taskWebSearchDo) Group(cols ...field.Expr) ITaskWebSearchDo {
	return t.withDO(t.DO.Group(cols...))
}

func (t taskWebSearchDo) Having(conds ...gen.Condition) ITaskWebSearchDo {
	return t.withDO(t.DO.Having(conds...))
}

func (t taskWebSearchDo) Limit(limit int) ITaskWebSearchDo {
	return t.withDO(t.DO.Limit(limit))
}

func (t taskWebSearchDo) Offset(offset int) ITaskWebSearchDo {
	return t.withDO(t.DO.Offset(offset))
}

func (t taskWebSearchDo) Scopes(funcs ...func(gen.Dao) gen.Dao) ITaskWebSearchDo {
	return t.withDO(t.DO.Scopes(funcs...))
}

func (t taskWebSearchDo) Unscoped() ITaskWebSearchDo {
	return t.withDO(t.DO.Unscoped())
}

func (t taskWebSearchDo) Create(values ...*model.TaskWebSearch) error {
	if len(values) == 0 {
		return nil
	}
	return t.DO.Create(values)
}

func (t taskWebSearchDo) CreateInBatches(values []*model.TaskWebSearch, batchSize int) error {
	return t.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (t taskWebSearchDo) Save(values ...*model.TaskWebSearch) error {
	if len(values) == 0 {
		return nil
	}
	return t.DO.Save(values)
}

func (t taskWebSearchDo) First() (*model.TaskWebSearch, error) {
	if result, err := t.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.TaskWebSearch), nil
	}
}

func (t taskWebSearchDo) Take() (*model.TaskWebSearch, error) {
	if result, err := t.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.TaskWebSearch), nil
	}
}

func (t taskWebSearchDo) Last() (*model.TaskWebSearch, error) {
	if result, err := t.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.TaskWebSearch), nil
	}
}

func (t taskWebSearchDo) Find() ([]*model.TaskWebSearch, error) {
	result, err := t.DO.Find()
	return result.([]*model.TaskWebSearch), err
}

func (t taskWebSearchDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.TaskWebSearch, err error) {
	buf := make([]*model.TaskWebSearch, 0, batchSize)
	err = t.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (t taskWebSearchDo) FindInBatches(result *[]*model.TaskWebSearch, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return t.DO.FindInBatches(result, batchSize, fc)
}

func (t taskWebSearchDo) Attrs(attrs ...field.AssignExpr) ITaskWebSearchDo {
	return t.withDO(t.DO.Attrs(attrs...))
}

func (t taskWebSearchDo) Assign(attrs ...field.AssignExpr) ITaskWebSearchDo {
	return t.withDO(t.DO.Assign(attrs...))
}

func (t taskWebSearchDo) Joins(fields ...field.RelationField) ITaskWebSearchDo {
	for _, _f := range fields {
		t = *t.withDO(t.DO.Joins(_f))
	}
	return &t
}

func (t taskWebSearchDo) Preload(fields ...field.RelationField) ITaskWebSearchDo {
	for _, _f := range fields {
		t = *t.withDO(t.DO.Preload(_f))
	}
	return &t
}

func (t taskWebSearchDo) FirstOrInit() (*model.TaskWebSearch, error) {
	if result, err := t.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.TaskWebSearch), nil
	}
}

func (t taskWebSearchDo) FirstOrCreate() (*model.TaskWebSearch, error) {
	if result, err := t.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.TaskWebSearch), nil
	}
}

func (t taskWebSearchDo) FindByPage(offset int, limit int) (result []*model.TaskWebSearch, count int64, err error) {
	result, err = t.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = t.Offset(-1).Limit(-1).Count()
	return
}

func (t taskWebSearchDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = t.Count()
	if err != nil {
		return
	}

	err = t.Offset(offset).Limit(limit).Scan(result)
	return
}

func (t taskWebSearchDo) Scan(result interface{}) (err error) {
	return t.DO.Scan(result)
}

func (t taskWebSearchDo) Delete(models ...*model.TaskWebSearch) (result gen.ResultInfo, err error) {
	return t.DO.Delete(models)
}

func (t *taskWebSearchDo) withDO(do gen.Dao) *taskWebSearchDo {
	t.DO = *do.(*gen.DO)
	return t
}
