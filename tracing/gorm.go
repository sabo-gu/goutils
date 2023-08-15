package tracing

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"

	"github.com/DoOR-Team/gorm"

	"github.com/json-iterator/go"
)

const (
	opentracingTracingOptionsKey    = "opentracingTracingOptionsKey"
	opentracingContextGormKey       = "opentracingContext"
	opentracingSpanStartTimeGormKey = "opentracingSpanStartTime"
)

// NewGormDB ...
func NewGormDB(ctx context.Context, db *gorm.DB) *gorm.DB {
	return db.Set(opentracingContextGormKey, ctx)
}

// NewGormDBWithTracingCallbacks ...
func NewGormDBWithTracingCallbacks(db *gorm.DB,
	options ...TracingOption) *gorm.DB {

	opts := &tracingOptions{}
	for _, opt := range options {
		opt(opts)
	}

	db = db.Set(opentracingTracingOptionsKey, opts)

	callbacks := newCallbacks()
	registerCallbacks(db, "create", callbacks)
	registerCallbacks(db, "batch_create", callbacks)
	registerCallbacks(db, "query", callbacks)
	registerCallbacks(db, "update", callbacks)
	registerCallbacks(db, "delete", callbacks)
	registerCallbacks(db, "row_query", callbacks)

	return db
}

func registerCallbacks(db *gorm.DB, name string, c *callbacks) {
	beforeName := fmt.Sprintf("opentracing:%v_before", name)
	afterName := fmt.Sprintf("opentracing:%v_after", name)

	gormCallbackName := fmt.Sprintf("gorm:%v", name)
	switch name {
	case "create":
		db.Callback().Create().Before(gormCallbackName).Register(beforeName, c.beforeCreate)
		db.Callback().Create().After(gormCallbackName).Register(afterName, c.afterCreate)
	case "batch_create":
		db.Callback().BatchCreate().Before(gormCallbackName).Register(beforeName, c.beforeBatchCreate)
		db.Callback().BatchCreate().After(gormCallbackName).Register(afterName, c.afterBatchCreate)
	case "query":
		db.Callback().Query().Before(gormCallbackName).Register(beforeName, c.beforeQuery)
		db.Callback().Query().After(gormCallbackName).Register(afterName, c.afterQuery)
	case "update":
		db.Callback().Update().Before(gormCallbackName).Register(beforeName, c.beforeUpdate)
		db.Callback().Update().After(gormCallbackName).Register(afterName, c.afterUpdate)
	case "delete":
		db.Callback().Delete().Before(gormCallbackName).Register(beforeName, c.beforeDelete)
		db.Callback().Delete().After(gormCallbackName).Register(afterName, c.afterDelete)
	case "row_query":
		db.Callback().RowQuery().Before(gormCallbackName).Register(beforeName, c.beforeRowQuery)
		db.Callback().RowQuery().After(gormCallbackName).Register(afterName, c.afterRowQuery)
	}
}

type callbacks struct{}

func newCallbacks() *callbacks {
	return &callbacks{}
}

func (c *callbacks) beforeCreate(scope *gorm.Scope)      { c.before(scope, "INSERT") }
func (c *callbacks) afterCreate(scope *gorm.Scope)       { c.after(scope, "INSERT") }
func (c *callbacks) beforeBatchCreate(scope *gorm.Scope) { c.before(scope, "BATCH_INSERT") }
func (c *callbacks) afterBatchCreate(scope *gorm.Scope)  { c.after(scope, "BATCH_INSERT") }
func (c *callbacks) beforeQuery(scope *gorm.Scope)       { c.before(scope, "SELECT") }
func (c *callbacks) afterQuery(scope *gorm.Scope)        { c.after(scope, "SELECT") }
func (c *callbacks) beforeUpdate(scope *gorm.Scope)      { c.before(scope, "UPDATE") }
func (c *callbacks) afterUpdate(scope *gorm.Scope)       { c.after(scope, "UPDATE") }
func (c *callbacks) beforeDelete(scope *gorm.Scope)      { c.before(scope, "DELETE") }
func (c *callbacks) afterDelete(scope *gorm.Scope)       { c.after(scope, "DELETE") }
func (c *callbacks) beforeRowQuery(scope *gorm.Scope)    { c.before(scope, "") }
func (c *callbacks) afterRowQuery(scope *gorm.Scope)     { c.after(scope, "") }

func (c *callbacks) before(scope *gorm.Scope, operation string) {
	//记录留作after使用
	scope.Set(opentracingSpanStartTimeGormKey, time.Now())
}

func (c *callbacks) after(scope *gorm.Scope, operation string) {
	val, ok := scope.Get(opentracingSpanStartTimeGormKey)
	if !ok {
		return
	}
	startTime, ok := val.(time.Time)
	if !ok {
		return
	}
	//用完即删
	scope.Set(opentracingSpanStartTimeGormKey, nil)

	//找到传递过来的ctx
	var ctx context.Context
	val, ok = scope.Get(opentracingContextGormKey)
	if ok {
		ctx, _ = val.(context.Context)
	}
	if ctx == nil {
		//如果为nil，我们就自己创建一个
		ctx = context.Background()
	}

	//处理ctx绑定tracing流
	if opentracing.SpanFromContext(ctx) == nil {
		//如果ctx里没存储span，就尝试从gls里获取
		glsSpan := getGlsTracingSpan()
		if glsSpan != nil {
			//塞入span
			ctx = opentracing.ContextWithSpan(ctx, glsSpan)
		}
	}

	//根据parentCtx，构造子span
	var parentCtx opentracing.SpanContext
	if parent := opentracing.SpanFromContext(ctx); parent != nil {
		parentCtx = parent.Context()
	}

	sql := strings.ToUpper(strings.TrimSpace(scope.SQL))

	op := operation
	if op == "" {
		//select count的单提出来，比较特殊
		if strings.Index(sql, "SELECT COUNT(") == 0 {
			op = "SELECT COUNT"
		} else {
			//这时候可以直接取sql头部作为method
			op = strings.Split(sql, " ")[0]
			if op == "" {
				op = "QUERY"
			}
		}
	}

	sp := opentracing.GlobalTracer().StartSpan(
		fmt.Sprintf("SQL %s %s", op, scope.TableName()),
		opentracing.ChildOf(parentCtx),
		opentracing.StartTime(startTime),
	)
	defer sp.Finish()

	//tag
	ext.Component.Set(sp, "sql")
	ext.DBType.Set(sp, scope.DB().Dialect().GetName())
	sp.SetTag("db.method", op)
	sp.SetTag("db.table", scope.TableName())

	//uid
	uid := sp.BaggageItem(BaggageItemKeyUserID)
	if uid != "" {
		sp.SetTag(TagKeyUserID, uid)
	}

	//记录请求
	values := []interface{}{scope.SQL, scope.SQLVars}
	query := getSQL(values...) //构造实际执行的sql语句
	sp.LogFields(log.String("request.body", query))

	//记录请求执行错误
	if scope.HasError() {
		err := scope.DB().Error

		//这个错误比较特殊，不是绝对意义上的错误，正常运行过程中比较常见，这里特殊处理
		if err == gorm.ErrRecordNotFound {
			sp.LogFields(log.String("event", err.Error()))
		} else {
			ext.Error.Set(sp, true)
			sp.LogFields(ErrorField(errors.Wrap(err, "Query failed")))
		}
	}

	opts := &tracingOptions{}
	val, ok = scope.DB().Get(opentracingTracingOptionsKey)
	if ok {
		opts = val.(*tracingOptions)
	}

	//记录结果
	result := &struct {
		RowsAffected    int64
		PrimaryKeyValue interface{} `json:",omitempty"`
		Result          interface{} `json:",omitempty"`
	}{
		RowsAffected:    scope.DB().RowsAffected,
		PrimaryKeyValue: scope.PrimaryKeyValue(),
	}

	if !opts.disableTracingGORMResultBody {
		if operation == "SELECT" {
			re := scope.Value
			if value, ok := scope.Get("gorm:query_destination"); ok {
				re = value
			}
			result.Result = re
		} else if operation == "" {
			////RowQuery 类型的
			//if re, ok := scope.InstanceGet("row_query_result"); ok {
			//	if _, ok := re.(*gorm.RowQueryResult); ok {
			//		//例如 Count(&xx) 会使用，这里就比较尴尬了，只能傻乎乎的通过调用栈去确认
			//		//if funcNameWithCaller(6) == "github.com/jinzhu/gorm.(*Scope).count" {
			//			//SELECT COUNT
			//
			//			//TODO: 这尴尬，Row.Scan方法执行过一次内部就会把Rows Close掉
			//			//所以外层就不能调用第二次了，需要再想招
			//
			//			// var cnt uint64
			//			// if err := rowResult.Row.Scan(&cnt); err != nil {
			//			// 	ext.Error.Set(sp, true)
			//			// 	sp.LogFields(ErrorField(errors.Wrap(err,"Scan row to cnt failed")))
			//			// } else {
			//			// 	result.Result = cnt
			//			// }
			//		//}
			//		// rowResult.Row //但是没有Row的ScanType的可用获取方式，不知道其类型不知道咋转
			//	}
			//	// else if rowsResult, ok := result.(*gorm.RowsQueryResult); ok {
			//	// 	rowResult.Rows //通过ScanType可以获取应该转的玩意，然后可以最终序列化出可读文本
			//	// }
			//}
		}
	}

	jsn, err := jsoniter.Marshal(result)
	if err != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(errors.Wrap(err, "Marshal result failed")))
	} else {
		sp.LogFields(log.String("result", pruneBodyLog(string(jsn), opts.maxBodyLogSize)))
	}
}

func funcNameWithCaller(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if ok {
		fun := runtime.FuncForPC(pc)
		return fun.Name()
	}
	return ""
}

var numericPlaceHolderRegexp = regexp.MustCompile(`\$\d+`)
var sqlRegexp = regexp.MustCompile(`\?`)

//TODO: 已知BUG -> 如果某值是枚举，转出来的会是枚举名称
func getSQL(values ...interface{}) string {
	var (
		sql             string
		formattedValues []string
	)
	for _, value := range values[1].([]interface{}) {
		indirectValue := reflect.Indirect(reflect.ValueOf(value))
		if indirectValue.IsValid() {
			value = indirectValue.Interface()
			if t, ok := value.(time.Time); ok {
				formattedValues = append(formattedValues, fmt.Sprintf("'%v'", t.Format("2006-01-02 15:04:05")))
			} else if b, ok := value.([]byte); ok {
				if str := string(b); isPrintable(str) {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", str))
				} else {
					formattedValues = append(formattedValues, "'<binary>'")
				}
			} else if r, ok := value.(driver.Valuer); ok {
				if value, err := r.Value(); err == nil && value != nil {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
				} else {
					formattedValues = append(formattedValues, "NULL")
				}
			} else {
				formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
			}
		} else {
			formattedValues = append(formattedValues, "NULL")
		}
	}

	// differentiate between $n placeholders or else treat like ?
	if numericPlaceHolderRegexp.MatchString(values[0].(string)) {
		sql = values[0].(string)
		for index, value := range formattedValues {
			placeholder := fmt.Sprintf(`\$%d([^\d]|$)`, index+1)
			sql = regexp.MustCompile(placeholder).ReplaceAllString(sql, value+"$1")
		}
	} else {
		formattedValuesLength := len(formattedValues)
		for index, value := range sqlRegexp.Split(values[0].(string), -1) {
			sql += value
			if index < formattedValuesLength {
				sql += formattedValues[index]
			}
		}
	}
	return sql
}

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
