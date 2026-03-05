package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WeeklyRotateWriter writes to log files that rotate on ISO-week boundaries.
// File naming: {prefix}-{year}-W{week}.log, e.g. app-2026-W09.log
type WeeklyRotateWriter struct {
	Dir    string
	Prefix string

	mu      sync.Mutex
	file    *os.File
	curWeek string
}

func (w *WeeklyRotateWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	year, week := time.Now().ISOWeek()
	weekStr := fmt.Sprintf("%d-W%02d", year, week)

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

func (w *WeeklyRotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
