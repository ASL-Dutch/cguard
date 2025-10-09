package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql" // 注册MySQL驱动
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"sysafari.com/customs/cguard/internal/config"
)

var (
	instance *sqlx.DB
	once     sync.Once
	mu       sync.RWMutex
)

// GetDB 返回数据库连接单例
func GetDB() *sqlx.DB {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("数据库未初始化，请先调用 InitDB")
	}

	return instance
}

// ExportDB 导出数据库连接供全局使用
// 此方法用于兼容旧代码，新代码应直接使用GetDB()
func ExportDB() *sqlx.DB {
	return GetDB()
}

// WithContext 在上下文中执行数据库操作
func WithContext(ctx context.Context, fn func(*sqlx.DB) error) error {
	db := GetDB()

	// 创建子上下文，添加超时控制
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 监听上下文取消
	errCh := make(chan error, 1)
	go func() {
		errCh <- fn(db)
	}()

	// 等待操作完成或上下文取消
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// BeginTx 开始事务并返回事务对象
func BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	db := GetDB()

	// 创建子上下文，添加超时控制
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	})
}

// InitDB 初始化数据库连接
func InitDB(cfg *config.MySQLConfig) error {
	var err error
	once.Do(func() {
		log.Info("初始化数据库连接...")
		db, dbErr := sqlx.Open("mysql", cfg.URL)
		if dbErr != nil {
			err = fmt.Errorf("连接数据库失败: %w", dbErr)
			return
		}

		// 设置连接池参数
		db.SetMaxOpenConns(cfg.MaxOpenConnections)
		db.SetMaxIdleConns(cfg.MaxIdleConnections)
		db.SetConnMaxLifetime(time.Duration(cfg.MaxLifeTime) * time.Minute)

		// 测试连接
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if dbErr = db.PingContext(ctx); dbErr != nil {
			err = fmt.Errorf("数据库连接测试失败: %w", dbErr)
			return
		}

		log.Info("数据库连接状态:", db.Stats())
		instance = db

		// 启动健康检查
		go startHealthCheck()
	})

	return err
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		log.Info("关闭数据库连接...")
		err := instance.Close()
		instance = nil
		return err
	}
	log.Debug("数据库连接未初始化，无需关闭")
	return nil
}

// 数据库健康检查
func startHealthCheck() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if instance == nil {
			log.Warn("数据库连接未初始化，健康检查跳过")
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := instance.PingContext(ctx)
		cancel()

		if err != nil {
			log.Error("数据库健康检查失败:", err)
		} else {
			log.Debug("数据库健康检查通过")
		}

		// 记录当前连接池状态
		log.Debug("数据库连接状态:", instance.Stats())
	}
}
