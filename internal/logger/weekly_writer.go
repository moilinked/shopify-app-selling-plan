package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// 每个 ISO 周生成一个文件，命名格式: {Prefix}-{年}-W{周}.log。
// 例: app-2026-W09.log
type WeeklyRotateWriter struct {
	Dir    string
	Prefix string

	mu      sync.Mutex
	file    *os.File
	curWeek string
}

// Write 写入日志内容。如果当前周发生变化，自动关闭旧文件并打开新文件。
func (w *WeeklyRotateWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	year, week := time.Now().ISOWeek()
	weekStr := fmt.Sprintf("%d-W%02d", year, week)

	// 周发生变化或首次写入时，切换到新的日志文件
	if weekStr != w.curWeek || w.file == nil {
		if w.file != nil {
			_ = w.file.Close()
		}
		filename := filepath.Join(w.Dir, fmt.Sprintf("%s-%s.log", w.Prefix, weekStr))
		w.file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return 0, err
		}
		w.curWeek = weekStr
	}

	return w.file.Write(p)
}

// Close 关闭当前打开的日志文件。
func (w *WeeklyRotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
