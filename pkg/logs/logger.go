package logs

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"

	ioz "github.com/hakadoriya/z.go/ioz"
)

//nolint:gochecknoglobals
var (
	Trace Logger = NewDiscard() //nolint:revive
	Debug Logger = NewDiscard() //nolint:revive
	Info  Logger = &DefaultLogger{log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)}
	Warn  Logger = &DefaultLogger{log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)}
)

func NewDiscard() Logger { //nolint:ireturn
	return &DefaultLogger{log.New(io.Discard, "", 0)}
}

func NewTrace() Logger { //nolint:ireturn
	return &DefaultLogger{log.New(os.Stderr, "TRACE: ", log.Ldate|log.Ltime|log.Lshortfile)}
}

func NewDebug() Logger { //nolint:ireturn
	return &DefaultLogger{log.New(os.Stderr, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)}
}

type Logger interface {
	io.Writer
	Print(v ...any)
	Printf(format string, v ...any)
	LineWriter(prefix string) io.Writer
}

const callerSkip = 2

type DefaultLogger struct {
	*log.Logger
}

func (l *DefaultLogger) Print(v ...any) { _ = l.Output(callerSkip, fmt.Sprint(v...)) }
func (l *DefaultLogger) Printf(format string, v ...any) {
	_ = l.Output(callerSkip, fmt.Sprintf(format, v...))
}

func (l *DefaultLogger) Write(p []byte) (n int, err error) {
	l.Print(string(p))
	return len(p), nil
}

func (l *DefaultLogger) LineWriter(prefix string) io.Writer {
	return ioz.WriteFunc(func(p []byte) (n int, err error) {
		for line := range bytes.SplitSeq(p, []byte("\n")) {
			_ = l.Output(1, prefix+string(line))
		}

		return len(p), nil
	})
}
