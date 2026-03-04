package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"os"
	"strings"
	"time"
)

type Logger struct {
	level string
	base  map[string]string
}

func New(level string) *Logger {
	return &Logger{level: strings.ToLower(level), base: map[string]string{"service": "file-engine"}}
}

func (l *Logger) cloneBase() map[string]any {
	m := map[string]any{}
	for k, v := range l.base {
		m[k] = v
	}
	return m
}

func (l *Logger) logStructured(level, message string, fields map[string]any) {
	entry := l.cloneBase()
	entry["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	entry["level"] = level
	entry["message"] = message
	maps.Copy(entry, fields)
	b, err := json.Marshal(entry)
	if err != nil {
		log.Printf("{\"level\":\"error\",\"message\":\"log marshal failed\",\"error\":%q}", err.Error())
		return
	}
	log.Println(string(b))
}

func (l *Logger) Event(level, message string, fields map[string]any) {
	if fields == nil {
		fields = map[string]any{}
	}
	switch level {
	case "debug":
		if l.level == "debug" {
			l.logStructured(level, message, fields)
		}
	default:
		l.logStructured(level, message, fields)
	}
}

func (l *Logger) Info(v ...any) { l.Event("info", fmt.Sprint(v...), nil) }
func (l *Logger) Infof(format string, v ...any) {
	l.Event("info", fmt.Sprintf(format, v...), nil)
}

func (l *Logger) Warnf(format string, v ...any) {
	l.Event("warn", fmt.Sprintf(format, v...), nil)
}
func (l *Logger) Fatal(v ...any) { l.Event("fatal", fmt.Sprint(v...), nil); os.Exit(1) }
func (l *Logger) Fatalf(format string, v ...any) {
	l.Event("fatal", fmt.Sprintf(format, v...), nil)
	os.Exit(1)
}
func (l *Logger) Debug(v ...any) { l.Event("debug", fmt.Sprint(v...), nil) }
func (l *Logger) Debugf(format string, v ...any) {
	l.Event("debug", fmt.Sprintf(format, v...), nil)
}
