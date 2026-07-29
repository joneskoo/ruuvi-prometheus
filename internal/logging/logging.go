package logging

import (
	"io/ioutil"
	"log"
	"os"
)

// Available loggers
var (
	Trace   *log.Logger
	Debug   *log.Logger
	Warning *log.Logger
)

func init() {
	Trace = log.New(ioutil.Discard, "TRACE ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(ioutil.Discard, "DEBUG ", log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(ioutil.Discard, "WARNING ", log.LstdFlags)
	SetLogLevel(DefaultLoggingLevel)
}

// Level is the type for logging level
type Level int

// Available logging levels
const (
	TRACE Level = iota
	DEBUG
	WARNING
	DISABLED
)

// DefaultLoggingLevel is the logging level we start with
const DefaultLoggingLevel Level = WARNING

// current logging level
var loggingLevel Level = DefaultLoggingLevel

// SetLogLevel sets the current logging level to given one
func SetLogLevel(lvl Level) {
	switch lvl {
	case TRACE:
		Trace.SetOutput(os.Stderr)
		Debug.SetOutput(os.Stderr)
		Warning.SetOutput(os.Stderr)
	case DEBUG:
		Trace.SetOutput(ioutil.Discard)
		Debug.SetOutput(os.Stderr)
		Warning.SetOutput(os.Stderr)
	case WARNING:
		Trace.SetOutput(ioutil.Discard)
		Debug.SetOutput(ioutil.Discard)
		Warning.SetOutput(os.Stderr)
	case DISABLED:
		Trace.SetOutput(ioutil.Discard)
		Debug.SetOutput(ioutil.Discard)
		Warning.SetOutput(ioutil.Discard)
	}
	loggingLevel = lvl
}

// GetLogLevel returns current log level
func GetLogLevel() Level {
	return loggingLevel
}

// IfTracing will call the given function with Trace logger if
// current logging level is to enable traces
// Allows to create time-consuming traces only when they are
// traced.
func IfTracing(f func(l *log.Logger)) {
	if loggingLevel == TRACE {
		f(Trace)
	}
}
