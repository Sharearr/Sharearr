package sharearr

import (
	"io"
	"log/slog"
	"os"

	slogc "github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
)

type prefixWriter struct {
	w      io.Writer
	prefix []byte
}

func (pw *prefixWriter) Write(p []byte) (int, error) {
	_, err := pw.w.Write(append(pw.prefix, p...))
	return len(p), err
}

func newLogWriter(w io.Writer) *prefixWriter {
	return &prefixWriter{w: w, prefix: []byte("[sharearr] ")}
}

func SetupLogger(cfg LogConfig) gin.HandlerFunc {
	level := parseLevel(cfg.Level)
	w := openOutput(cfg)
	l := newLogger(w, level)
	slog.SetDefault(l)
	return slogc.SetLogger(ginSlogOptions(w, l, level)...)
}

func parseLevel(s string) slog.Level {
	var level slog.Level
	if err := level.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelInfo
	}
	return level
}

func openOutput(cfg LogConfig) io.Writer {
	pw := newLogWriter(os.Stdout)
	if cfg.File == "" {
		return pw
	}
	f, err := os.OpenFile(cfg.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Failed to open log file", "file", cfg.File, "error", err)
		panic("failed to open log file")
	}
	gin.DefaultWriter = io.MultiWriter(os.Stdout, f)
	gin.DefaultErrorWriter = io.MultiWriter(os.Stderr, f)
	return io.MultiWriter(pw, newLogWriter(f))
}

func newLogger(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))
}

func ginSlogOptions(w io.Writer, l *slog.Logger, level slog.Level) []slogc.Option {
	return []slogc.Option{
		slogc.WithWriter(w),
		slogc.WithLogger(func(_ *gin.Context, _ *slog.Logger) *slog.Logger {
			return l
		}),
		slogc.WithDefaultLevel(level),
		slogc.WithContext(func(c *gin.Context, rec *slog.Record) *slog.Record {
			if res, ok := c.Get("count"); ok {
				rec.Add("count", res)
			}
			return rec
		}),
	}
}
