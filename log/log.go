package log

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/fatih/color"
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
	h, _ := NewStreamHandler( /*os.Stdout*/ color.Output)
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

func (l *Logger) Output(callDepth int, level LogLever, format string, v ...interface{}) {
	if l.level&level != level {
		return
	}

	buf := l.popBuf()

	buf = append(buf, time.Now().Format(TimeFormat)...)
	buf = append(buf, " - "...)

	buf = append(buf, l.colorLevel(level)...)
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

	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf = append(buf, '\n')
	}

	l.msg <- buf
}

func (l *Logger) colorLevel(level LogLever) string {

	switch level {
	case LevelDebug:
	case LevelInfo:
		return color.GreenString(levelName(level))
	case LevelWarn:
		return color.YellowString(levelName(level))
	case LevelError:
		return color.RedString(levelName(level))
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
	l.Output(2, LevelDebug, format, v...)
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.Output(2, LevelInfo, format, v...)
}

func (l *Logger) Warn(format string, v ...interface{}) {
	l.Output(2, LevelWarn, format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.Output(2, LevelError, format, v...)
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
	defLoger.Output(2, LevelDebug, format, v...)
}

func DebugLine(format string, v ...interface{}) {
	defLoger.Output(2, LevelDebug, format, v...)
}

func Info(format string, v ...interface{}) {
	defLoger.Output(2, LevelInfo, format, v...)
}

func Warn(format string, v ...interface{}) {
	defLoger.Output(2, LevelWarn, format, v...)
}

func Error(format string, v ...interface{}) {
	defLoger.Output(2, LevelError, format, v...)
}

func Error2(err error) {
	defLoger.Output(2, LevelError, "%v", err)
}

func StdLogger() *Logger {
	return defLoger
}

func GetLevel() LogLever {
	return defLoger.level
}

func init() {
	SetLevel(LevelAll)
}
