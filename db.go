package ginsass

import (
	"database/sql"
	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"strconv"
	"time"
)

var (
	db *DB
)

type DB struct {
	*gorm.DB
}

func initDb() {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer（日志输出的目标，前缀和日志包含的内容——译者注）
		logger.Config{
			SlowThreshold:             time.Second, // 慢 SQL 阈值
			LogLevel:                  logger.Info, // 日志级别
			IgnoreRecordNotFoundError: true,        // 忽略ErrRecordNotFound（记录未找到）错误
			Colorful:                  false,       // 禁用彩色打印
		},
	)

	dsn := GlobalConfig().DbConfig.ToDsn()

	if dsn == "" {
		logrus.Warnf("跳过连接数据库")
		return
	}

	logrus.Infof("开始连接数据库: %v", dsn)

	switch GlobalConfig().DbConfig.Driver {
	case "sqlite":
		d, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: newLogger})
		if err != nil {
			logrus.WithError(err).Warnf("连接SQLITE失败: %v", dsn)
			return
		}
		db = &DB{d}
	case "mysql":
		sqlDB, err := sql.Open("mysql", dsn)
		if err != nil {
			logrus.WithError(err).Warnf("连接 mysql 失败: %v", dsn)
			return
		}
		gormDB, err := gorm.Open(mysql.New(mysql.Config{
			Conn: sqlDB,
		}), &gorm.Config{Logger: newLogger})
		if err != nil {
			logrus.WithError(err).Warnf("连接 mysql 失败 %v", dsn)
			return
		}
		db = &DB{gormDB}
	case "postgres":
		d, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: newLogger})
		if err != nil {
			logrus.WithError(err).Warnf("连接postgres失败: %v", dsn)
			return
		}
		db = &DB{d}
	default:
		logrus.Warnf("跳过连接数据库, 不支持的驱动: %v", GlobalConfig().DbConfig.Driver)
	}
	if db == nil {
		return
	}
}

func GetDB() *DB {
	return db
}

func Paginate(ctx *Context, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		q := ctx.Request.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		if page == 0 {
			page = 1
		}
		ctx.Assign("page", page)
		ctx.Assign("pageSize", pageSize)
		qr := ctx.Request.URL.Query()
		qr.Del("page")
		ctx.Assign("query", qr.Encode())
		switch {
		case pageSize > 100:
			pageSize = 100
		case pageSize <= 0:
			pageSize = 12
		}
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

func FindAndCount(ctx *Context, pageSize int, db *gorm.DB, rows interface{}, count *int64) error {
	a := db.Count(count).Scopes(Paginate(ctx, pageSize)).Find(rows)
	if a.Error != nil {
		return a.Error
	}
	return nil
}

func Random(db *gorm.DB, limit int, rows interface{}) error {
	tx := db.Order("random()").Limit(limit).Find(rows)
	return tx.Error
}
