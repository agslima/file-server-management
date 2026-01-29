package logger

import (
    "log"
    "strings"
)

type Logger struct {
    level string
}

func New(level string) *Logger {
    return &Logger{level: strings.ToLower(level)}
}

func (l *Logger) Info(v ...interface{}) { log.Println(v...) }
func (l *Logger) Infof(format string, v ...interface{}) { log.Printf(format, v...) }
func (l *Logger) Fatal(v ...interface{}) { log.Fatalln(v...) }
func (l *Logger) Fatalf(format string, v ...interface{}) { log.Fatalf(format, v...) }
func (l *Logger) Debug(v ...interface{}) { if l.level == "debug" { log.Println(v...) } }
func (l *Logger) Debugf(format string, v ...interface{}) { if l.level == "debug" { log.Printf(format, v...) } }
