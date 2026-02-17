package logging

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"
)

// GormLogger adapts zap.Logger for GORM's logger interface.
// It logs SQL queries at debug level and errors at error level.
type GormLogger struct {
	logger *zap.Logger
	level  gormlogger.LogLevel
}

// NewGormLogger creates a GORM logger backed by zap.
func NewGormLogger(logger *zap.Logger) gormlogger.Interface {
	return &GormLogger{
		logger: logger.Named("gorm"),
		level:  gormlogger.Warn,
	}
}

func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &GormLogger{logger: l.logger, level: level}
}

func (l *GormLogger) Info(_ context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Info {
		l.logger.Sugar().Infof(msg, args...)
	}
}

func (l *GormLogger) Warn(_ context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Warn {
		l.logger.Sugar().Warnf(msg, args...)
	}
}

func (l *GormLogger) Error(_ context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Error {
		l.logger.Sugar().Errorf(msg, args...)
	}
}

func (l *GormLogger) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && !errors.Is(err, gormlogger.ErrRecordNotFound):
		l.logger.Error("query error",
			zap.Error(err),
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	case elapsed > 200*time.Millisecond:
		l.logger.Warn("slow query",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	default:
		l.logger.Debug("query",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	}
}
