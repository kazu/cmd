package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	//"ghe.justice-tech.com/KOMABA/csl-etl/ext"
	"log/slog"

	util "github.com/kazu/cmd/ext"
)

type OptFunc[T any] func(*T)

func HandleOpt[T any, O OptFunc[T]](arg *T, opts ...O) {

	for _, optfn := range opts {
		optfn(arg)
	}
}

type logger struct {
	logger        *slog.Logger
	level         slog.LevelVar
	setHandleropt func(opt *slog.HandlerOptions) slog.Handler
}

var currentLogger logger
var currentLoggerMu sync.Mutex

func Handler(fn func(opt *slog.HandlerOptions) slog.Handler) OptFunc[logger] {
	return func(l *logger) {
		l.setHandleropt = fn
	}
}

func SimpleHander(f io.Writer) func(opt *slog.HandlerOptions) slog.Handler {

	return func(opt *slog.HandlerOptions) slog.Handler {
		return slog.NewTextHandler(f, opt)
	}
}
func DefaultHandler() func(opt *slog.HandlerOptions) slog.Handler {
	return func(opt *slog.HandlerOptions) slog.Handler {
		return SimpleHander(os.Stdout)(opt)
	}
}

func SetHandler[T any](f io.Writer, handler func(io.Writer, *slog.HandlerOptions) T) func(opt *slog.HandlerOptions) slog.Handler {

	return func(opt *slog.HandlerOptions) slog.Handler {
		return (any)(handler(f, opt)).(slog.Handler)
	}
}

var defaulttwriter strings.Builder

func DefaultWriter() *strings.Builder {
	return &defaulttwriter
}

var _handlerOpt slog.HandlerOptions

func SetHandlerOpt(opt slog.HandlerOptions) {
	_handlerOpt = opt
}

func Init(l *logger) {

	if l.setHandleropt == nil {
		l.setHandleropt = SimpleHander(&defaulttwriter)
	}
	opt := &_handlerOpt
	opt.Level = &l.level

	l.logger = slog.New(l.setHandleropt(opt))
	slog.SetDefault(l.logger)

}

var _enabled = false

var Logger = util.OnceArgsValue(_Logger)

func _Logger(opts ...OptFunc[logger]) *logger {

	HandleOpt(&currentLogger, opts...)
	if currentLogger.logger == nil || len(opts) > 0 {
		Init(&currentLogger)
	}
	_enabled = true
	return &currentLogger
}

func Level(lvl slog.Level) func(l *logger) {

	return func(l *logger) {
		if l.level.Level().Level() == lvl {
			return
		}
		l.level.Set(lvl)
	}

}

func LazyAttr(attrs *[]slog.Attr) func(adds ...slog.Attr) {

	return func(adds ...slog.Attr) {

		*attrs = append(*attrs, adds...)

	}

}

type LazyFunc func(adds ...slog.Attr)
type LazyHandler func(attr LazyFunc)

func logable(logger *slog.Logger) func(mes string, level slog.Level, fn LazyHandler) {

	attrs := []slog.Attr{}
	lattrfn := LazyAttr(&attrs)

	return func(mes string, level slog.Level, fn LazyHandler) {

		fn(lattrfn)

		logger.LogAttrs(
			context.Background(),
			level,
			mes,
			attrs...)
	}
}

func emptyFn(fn LazyFunc) {
}

func InfoM(mes string) {
	Info(mes)(emptyFn)
}

func DebugM(mes string) {
	Debug(mes)(emptyFn)
}

func WarnM(mes string) {
	Debug(mes)(emptyFn)
}

func ErrorM(mes string) {
	Debug(mes)(emptyFn)
}

type withLevelFn func(string) func(fn LazyHandler)

func Info(mes string) func(fn LazyHandler) {
	return Lazy(slog.LevelInfo)(mes)
}

func Debug(mes string) func(fn LazyHandler) {
	return Lazy(slog.LevelDebug)(mes)
}
func Error(mes string) func(fn LazyHandler) {
	return Lazy(slog.LevelError)(mes)
}
func Warn(mes string) func(fn LazyHandler) {
	return Lazy(slog.LevelWarn)(mes)
}

func emptyLazy(mes string) func(fn LazyHandler) { return func(fn LazyHandler) {} }

func Lazy(level slog.Level) func(mes string) func(fn LazyHandler) {
	if !_enabled {
		return emptyLazy
	}
	l := Logger()
	if l.level.Level() > level {
		return emptyLazy
	}

	return func(mes string) func(fn LazyHandler) {

		return func(fn LazyHandler) {
			logable(l.logger)(mes, level, fn)
		}
	}
}

func WithErr(e error) slog.Attr {
	if e == nil {
		return slog.String("err", "nil")
	}
	return slog.String("err", e.Error())
}

type logAttr interface {
	string | int | int64 | uint64 | time.Time | time.Duration | float64
}

func type2Attr[T logAttr](idx int, v T) slog.Attr {
	switch v := (interface{})(v).(type) {
	case string:
		return slog.String(fmt.Sprintf("%d", idx), v)
	case int:
		return slog.Int(fmt.Sprintf("%d", idx), v)
	case int64:
		return slog.Int64(fmt.Sprintf("%d", idx), v)
	case uint64:
		return slog.Uint64(fmt.Sprintf("%d", idx), v)
	case float64:
		return slog.Float64(fmt.Sprintf("%d", idx), v)
	case time.Time:
		return slog.Time(fmt.Sprintf("%d", idx), v)
	case time.Duration:
		return slog.Duration(fmt.Sprintf("%d", idx), v)
	}

	return slog.Attr{}
}

func Slice[T logAttr](key string, slices []T) slog.Attr {

	attrs := make([]any, 0, len(slices))

	for idx, slice := range slices {
		attrs = append(attrs, type2Attr(idx, slice))
	}
	return slog.Group(key, attrs...)
}

func String2level(str string) slog.Level {

	l, found := map[string]slog.Level{
		"info":  slog.LevelInfo,
		"debug": slog.LevelDebug,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError}[str]
	if found {
		return l
	}

	return slog.LevelInfo

}

func OnlyDebug(fn func()) {

	Debug("empty message")(func(a LazyFunc) {
		fn()
	})
}

type loggerOpt struct {
	w          func(name string) io.Writer
	handlerOpt *slog.HandlerOptions
}

func findSimpleWriter(logName string) (w io.Writer) {

	if logName == "stdout" {
		w = os.Stdout
	}
	if logName == "stderr" {
		w = os.Stderr
	}
	if logName == "null" {
		w = io.Discard
	}
	if w == os.Stderr && len(logName) > 0 {
		os.MkdirAll(filepath.Dir(logName), 0750)
		wr, e := os.OpenFile(logName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if e == nil {
			w = wr
		}
	}
	return
}

func SetupLogger(logName string, loglevel string, opts ...util.OptFunc[loggerOpt]) {
	var w io.Writer
	w = os.Stderr
	opt := loggerOpt{w: findSimpleWriter}
	util.HandleOpt(&opt, opts...)

	w = opt.w(logName)
	if opt.handlerOpt != nil {
		SetHandlerOpt(*opt.handlerOpt)
	}

	Logger(
		Level(String2level(loglevel)),
		Handler(SimpleHander(w)))
}
