// Package logger provides a simple logging implementation to be used in conjunction with Puff.
package puff

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"runtime"
	"time"

	"github.com/ThePuffProject/puff/color"
)

// LoggerConfig is used to dictate logger behavior.
type LoggerConfig struct {
	// UseJSON will enable/disable JSON mode for the logger.
	UseJSON bool
	// Indent will control whether to use MarshalIndent or Marshal for logging.
	Indent bool
	// Level is the logger level. Logger will only write to stdout if log record is above or equivalent to Level.
	Level slog.Level
	// TimeFormat is the textual representation of the timestamp. It is passed as an argument to time.Format()
	TimeFormat string
	// AddSource is equivalent to slog.HandlerOptions.AddSource
	AddSource bool
	// Colorize enables or disables pretty logging dependant on LogLevel.
	Colorize bool
}

var DefaultLoggerConfig = LoggerConfig{
	UseJSON:    false,
	Level:      slog.LevelInfo,
	TimeFormat: time.UnixDate,
	Colorize:   true,
}

var DefaultJSONLoggerConfig = LoggerConfig{
	UseJSON:    true,
	Level:      slog.LevelDebug,
	TimeFormat: time.StampNano,
	Colorize:   false,
}

// SlogHandler is puff's implementation of structured logging.
// It wraps golang's slog package.
type SlogHandler struct {
	slog.Handler
	config LoggerConfig
}

// NewSlogHandler returns a new puff.SlogHandler given a LoggerConfig and slog.Handler
func NewSlogHandler(config LoggerConfig) *SlogHandler {
	return &SlogHandler{
		Handler: &slog.TextHandler{},
		config:  config,
	}
}

// Enabled will check if a log needs to be written to stdout.
func (h *SlogHandler) Enabled(c context.Context, level slog.Level) bool {
	return level >= h.config.Level
}

// Handle will write to stdout.
func (h *SlogHandler) Handle(c context.Context, r slog.Record) error {
	context.Background()
	level := r.Level.String()
	// level_formatted := fmt.Sprintf("%s:", r.Level.String())

	if h.config.Colorize {
		switch r.Level {
		case slog.LevelDebug:
			level = color.ColorizeBold(level, color.FgMagenta)
		case slog.LevelInfo:
			level = color.ColorizeBold(level, color.FgBlue)
		case slog.LevelWarn:
			level = color.ColorizeBold(level, color.FgYellow)
		case slog.LevelError:
			level = color.ColorizeBold(level, color.FgRed)
		}
	}

	fields := make(map[string]any, r.NumAttrs())
	// populate fields
	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.Any()
		return true
	})
	if h.config.AddSource {
		fields["source"] = createSource(r.PC)
	}
	timeStr := r.Time.Format(h.config.TimeFormat)

	var attrs_formatted []byte
	var err error

	if h.config.UseJSON {
		fields["level"] = level
		fields["time"] = timeStr
		fields["message"] = r.Message
	}
	if h.config.Indent {
		attrs_formatted, err = json.MarshalIndent(fields, "", "  ")
		if err != nil {
			return err
		}
	} else {
		attrs_formatted, err = json.Marshal(fields)
		if err != nil {
			return err
		}
	}

	if h.config.UseJSON {
		fmt.Println(string(attrs_formatted))
		return nil
	}

	if len(fields) > 0 {
		fmt.Println(timeStr, fmt.Sprintf("%s:", level), r.Message, string(attrs_formatted))
	} else {
		fmt.Println(timeStr, fmt.Sprintf("%s:", level), r.Message)
	}
	return nil
}

// SetLevel changes the puff.SlogHandler level to the one specified.
func (h *SlogHandler) SetLevel(level slog.Level) {
	h.config.Level = level
}

// NewLogger creates a new *slog.Logger provided the LoggerConfig.
// Use this function if the default loggers; DefaultLogger and DefaultJSONLogger are not satisfactory.
func NewLogger(c *LoggerConfig) *slog.Logger {
	if c.Colorize && c.UseJSON {
		panic("Cannot enable both json and color mode. Please pick only one.")
	}
	return slog.New(NewSlogHandler(*c))
}

func createSource(pc uintptr) *slog.Source {

	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	return &slog.Source{
		Function: f.Function,
		File:     path.Base(f.File),
		Line:     f.Line,
	}
}

// DefaultLogger returns a slog.Logger which will use the default text logger.
func DefaultLogger() *slog.Logger {
	return NewLogger(&DefaultLoggerConfig)
}

// DefaultLogger returns a slog.Logger which will use the default json logger.
func DefaultJSONLogger() *slog.Logger {
	return NewLogger(&DefaultJSONLoggerConfig)
}
