package log

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type LogLever int

const (
	LevelInfo  LogLever = 1
	LevelDebug LogLever = 1 << 1
	LevelWarn  LogLever = 1 << 2
	LevelError LogLever = 1 << 3
	LevelAll   LogLever = LevelInfo | LevelDebug | LevelWarn | LevelError
)

const TimeFormat = "2006/01/02 15:04:05"

const maxBufPoolSize = 16

const (
	NONE         = "\033[m"
	RED          = "\033[0;32;31m"
	LIGHT_RED    = "\033[1;31m"
	GREEN        = "\033[0;32;32m"
	LIGHT_GREEN  = "\033[1;32m"
	BLUE         = "\033[0;32;34m"
	LIGHT_BLUE   = "\033[1;34m"
	DARY_GRAY    = "\033[1;30m"
	CYAN         = "\033[0;36m"
	LIGHT_CYAN   = "\033[1;36m"
	PURPLE       = "\033[0;35m"
	LIGHT_PURPLE = "\033[1;35m"
	BROWN        = "\033[0;33m"
	YELLOW       = "\033[1;33m"
	LIGHT_GRAY   = "\033[0;37m"
	WHITE        = "\033[1;37m"
)

type Logger struct {
	sync.Mutex

	level LogLever
	flag  int

	handler Handler

	quit chan struct{}
	msg  chan []byte

	bufs [][]byte

	wg sync.WaitGroup

	closed bool
}

func New(handler Handler) *Logger {
	var l = new(Logger)

	l.level = LevelInfo
	l.handler = handler

	l.quit = make(chan struct{})
	l.closed = false

	l.msg = make(chan []byte, 1024)

	l.bufs = make([][]byte, 0, 16)

	l.wg.Add(1)
	go l.run()

	return l
}

func NewDefault(handler Handler) *Logger {
	return New(handler)
}

func newStdHandler() *StreamHandler {
	h, _ := NewStreamHandler(os.Stdout)
	return h
}

var defLoger = NewDefault(newStdHandler())

func Close() {
	defLoger.Close()
}

func (l *Logger) run() {
	defer l.wg.Done()
	for {
		select {
		case msg := <-l.msg:
			l.handler.Write(msg)
			l.putBuf(msg)
		case <-l.quit:
			if len(l.msg) == 0 {
				return
			}
		}
	}
}

func (l *Logger) popBuf() []byte {
	l.Lock()
	var buf []byte
	if len(l.bufs) == 0 {
		buf = make([]byte, 0, 1024)
	} else {
		buf = l.bufs[len(l.bufs)-1]
		l.bufs = l.bufs[0 : len(l.bufs)-1]
	}
	l.Unlock()

	return buf
}

func (l *Logger) putBuf(buf []byte) {
	l.Lock()
	if len(l.bufs) < maxBufPoolSize {
		buf = buf[0:0]
		l.bufs = append(l.bufs, buf)
	}
	l.Unlock()
}

func (l *Logger) Close() {
	if l.closed {
		return
	}
	l.closed = true

	close(l.quit)
	l.wg.Wait()
	l.quit = nil

	l.handler.Close()
}

func (l *Logger) SetLevel(level LogLever) {
	l.level = level
}

func (l *Logger) Level() LogLever {
	return l.level
}

func (l *Logger) Output(ln bool, callDepth int, level LogLever, format string, v ...interface{}) {
	if l.level&level != level {
		return
	}

	buf := l.popBuf()

	now := time.Now().Format(TimeFormat)
	if ln {
		buf = append(buf, "\033[1A"...)
	}
	buf = append(buf, now...)
	buf = append(buf, " - "...)

	buf = append(buf, l.colorStart(level)...)
	buf = append(buf, levelName(level)...)
	buf = append(buf, l.colorStop(level)...)
	buf = append(buf, " - "...)

	/*pc*/
	_, file, line, ok := runtime.Caller(callDepth)
	if !ok {
		file = "???"
		line = 0
	} else {
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				file = file[i+1:]
				break
			}
		}
	}

	buf = append(buf, file...)
	buf = append(buf, ":["...)

	buf = strconv.AppendInt(buf, int64(line), 10)
	buf = append(buf, "] - "...)

	/*if pc != 0 {
		f := runtime.FuncForPC(pc)
		if f != nil {
			funcNamePath := strings.Split(f.Name(), "/")
			buf = append(buf, "["+funcNamePath[len(funcNamePath)-1]+"] - "...)
		}
	}*/

	s := fmt.Sprintf(format, v...)

	buf = append(buf, s...)

	if ln {
		buf = append(buf, "\033[K"...)
	}

	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf = append(buf, '\n')
	}

	l.msg <- buf
}

func (l *Logger) colorStart(level LogLever) string {

	switch level {
	case LevelDebug:
	case LevelInfo:
		return GREEN
	case LevelWarn:
		return YELLOW
	case LevelError:
		return RED
	}
	return ""
}

func (l *Logger) colorStop(level LogLever) string {
	switch level {
	case LevelDebug:
	case LevelInfo:
		return NONE
	case LevelWarn:
		return NONE
	case LevelError:
		return NONE
	}
	return ""
}

func levelName(level LogLever) string {
	switch {
	case level&LevelDebug == LevelDebug:
		return "DEBUG"
	case level&LevelInfo == LevelInfo:
		return "INFO"
	case level&LevelWarn == LevelWarn:
		return "WARN"
	case level&LevelError == LevelError:
		return "ERROR"
	}
	return ""
}

func (l *Logger) Debug(format string, v ...interface{}) {
	l.Output(false, 2, LevelDebug, format, v...)
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.Output(false, 2, LevelInfo, format, v...)
}

func (l *Logger) Warn(format string, v ...interface{}) {
	l.Output(false, 2, LevelWarn, format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.Output(false, 2, LevelError, format, v...)
}

func SetLevel(level LogLever) {
	defLoger.SetLevel(level)
}

func SetLogFile(logFile string) {
	if defLoger != nil {
		defLoger.Close()
	}

	var h Handler

	if len(logFile) != 0 {
		var err error
		h, err = NewTimeRotatingFileHandler(logFile, WhenDay, 1)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	if h == nil {
		h = newStdHandler()
	}

	defLoger = NewDefault(h)
}

func Debug(format string, v ...interface{}) {
	defLoger.Output(false, 2, LevelDebug, format, v...)
}

func DebugLine(format string, v ...interface{}) {
	defLoger.Output(true, 2, LevelDebug, format, v...)
}

func Info(format string, v ...interface{}) {
	defLoger.Output(false, 2, LevelInfo, format, v...)
}

func Warn(format string, v ...interface{}) {
	defLoger.Output(false, 2, LevelWarn, format, v...)
}

func Error(format string, v ...interface{}) {
	defLoger.Output(false, 2, LevelError, format, v...)
}

func StdLogger() *Logger {
	return defLoger
}

func GetLevel() LogLever {
	return defLoger.level
}
