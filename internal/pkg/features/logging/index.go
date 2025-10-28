package logging

import (
	"goapp/pkg/db"
	"goapp/pkg/ids"
	"time"
)

type Level string

const (
	DebugLevel Level = "DEBUG"
	InfoLevel  Level = "INFO"
	WarnLevel  Level = "WARN"
	ErrorLevel Level = "ERROR"
	FatalLevel Level = "FATAL"
)

type ServiceLog struct {
	ID         ids.UID   `bun:"id,pk" json:"id"`
	Service    string    `bun:"service" json:"service"`
	TraceID    ids.UID   `bun:"trace_id" json:"traceId"`
	Level      Level     `bun:"level" json:"level"`
	Message    string    `bun:"message" json:"message"`
	StackTrace string    `bun:"stack_trace" json:"stackTrace"`
	Data       db.JSON   `bun:"data" json:"data"`
	CreatedAt  time.Time `bun:"created_at" json:"createdAt"`
}

type LogOption func(log *ServiceLog)

func WithTraceID(traceID ids.UID) LogOption {
	return func(log *ServiceLog) {
		log.TraceID = traceID
	}
}

func WithData(data db.JSON) LogOption {
	return func(log *ServiceLog) {
		log.Data = data
	}
}

func WithStackTrace(stackTrace string) LogOption {
	return func(log *ServiceLog) {
		log.StackTrace = stackTrace
	}
}

func doLog(level Level, msg string, options ...LogOption) {
	log := &ServiceLog{Level: level, Message: msg}
	for _, opt := range options {
		opt(log)
	}
	post(log)
}

func Debug(msg string, options ...LogOption) {
	doLog(DebugLevel, msg, options...)
}

func Info(msg string, options ...LogOption) {
	doLog(InfoLevel, msg, options...)
}

func Warn(msg string, options ...LogOption) {
	doLog(WarnLevel, msg, options...)
}

func Error(msg string, options ...LogOption) {
	doLog(ErrorLevel, msg, options...)
}

func Fatal(msg string, options ...LogOption) {
	doLog(FatalLevel, msg, options...)
}
