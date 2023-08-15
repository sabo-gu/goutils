package db

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/DoOR-Team/gorm"

	_ "github.com/DoOR-Team/gorm/dialects/mysql"

	"github.com/DoOR-Team/goutils/log"
	"github.com/DoOR-Team/goutils/tracing"
	"github.com/DoOR-Team/goutils/waitgroup"
)

func GetMySqlDBWithoutResultFromVipper(options ...tracing.TracingOption) *gorm.DB {
	return GetMySqlDBFromVipper(tracing.TracingGORMResultBody(false))
}

// tracing.TracingGORMResultBody(false)
func GetMySqlDBFromVipper(options ...tracing.TracingOption) *gorm.DB {
	dbname := viper.GetString("db_name")
	host := viper.GetString("db_host")
	user := viper.GetString("db_user")
	passwd := viper.GetString("db_password")

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True",
		user,
		passwd,
		host,
		dbname)

	log.Infof("开始初始化 MySQL 数据库连接%s", dsn)
	DB, err := gorm.Open("mysql", dsn+"&parseTime=True&loc=Local")
	if err != nil {
		log.Warn("MySQL 数据库连接错误，尝试使用无表状态连接", err)
		dsnWithoutDb := fmt.Sprintf("%s:%s@tcp(%s)/mysql?charset=utf8mb4&parseTime=True",
			user,
			passwd,
			host)
		DB, err = gorm.Open("mysql", dsnWithoutDb+"&parseTime=True&loc=Local")
		if err != nil {
			log.Fatal(err)
		}
		DB.LogMode(true)
		// db_name为官方默认库名的key
		createDB := DB.Exec("CREATE DATABASE IF NOT EXISTS " + viper.GetString("db_name") +
			" DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci")
		if createDB.Error != nil {
			// log.Println(createDB.Error)
			log.Fatal(createDB.Error)
		}
		DB.Close()

		log.Info("重试尝试连接 MySQL")
		DB, err = gorm.Open("mysql", dsn+"&parseTime=True&loc=Local")
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Info("MySQL 数据库连接初始化成功")

	// 配置一个日志格式的前缀
	// DB.SetLogger(Logger{})
	// DB.LogMode(logEnable)
	DB.LogMode(false)

	if tracing.Enable {
		// 5m限制
		opts := []tracing.TracingOption{tracing.TracingMaxBodyLogSize(5 * 1024 * 1024)}
		opts = append(opts, options...)
		DB = tracing.NewGormDBWithTracingCallbacks(DB, opts...)
		log.Println("gorm 注入tracing模块")
	}

	waitgroup.AddModAndWrapServer("GORM_Client", &waitgroup.Cli{
		CloseFunc: func() error {
			return DB.Close()
		},
	})

	// 这里一定要创建一个函数变量
	// queryfunc := gormCallBackTraceMidWay(true, DB.Callback().Query().Get("gorm:query"))
	// DB.Callback().Query().Replace("gorm:query", queryfunc)
	//
	// createfunc := gormCallBackTraceMidWay(false, DB.Callback().Create().Get("gorm:create"))
	// DB.Callback().Create().Replace("gorm:create", createfunc)
	//
	// updatefunc := gormCallBackTraceMidWay(false, DB.Callback().Update().Get("gorm:update"))
	// DB.Callback().Update().Replace("gorm:update", updatefunc)
	//
	// deletefunc := gormCallBackTraceMidWay(false, DB.Callback().Delete().Get("gorm:delete"))
	// DB.Callback().Delete().Replace("gorm:delete", deletefunc)
	//
	// rowfunc := gormCallBackTraceMidWay(false, DB.Callback().RowQuery().Get("gorm:row_query"))
	// DB.Callback().Delete().Replace("gorm:row_query", rowfunc)
	return DB
}
