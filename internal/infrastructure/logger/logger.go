package logger

import (
	"github.com/xvq/go-template/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

func New(c config.Log, name string) (*slog.Logger, func() error, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.Level)); err != nil {
		return nil, nil, err
	}
	var w io.Writer = os.Stdout
	closeFn := func() error { return nil }
	if c.Output != "stdout" {
		if err := os.MkdirAll(c.Dir, 0750); err != nil {
			return nil, nil, err
		}
		file := &lumberjack.Logger{Filename: filepath.Join(c.Dir, name+".log"), MaxSize: c.MaxSizeMB, MaxBackups: c.MaxBackups, MaxAge: c.MaxAgeDays, Compress: c.Compress}
		closeFn = file.Close
		w = file
		if c.Output == "both" {
			w = io.MultiWriter(os.Stdout, file)
		}
	}
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})).With("process", name), closeFn, nil
}
