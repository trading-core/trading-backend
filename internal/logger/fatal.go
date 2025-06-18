package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	na = "n/a"

	// Simple control of provenance output:

	// SimpleFormat dir/file:line information
	SimpleFormat = "(%[2]s/%[3]s:%[4]d)"

	// FullFormat function in dir/file:line information
	FullFormat = "(from %[1]s in %[2]s/%[3]s:%[4]d)"
)

var levels = map[string]bool{
	"TRACE":  envBool("LOG_TRACE", false),
	"METHOD": envBool("LOG_METHOD", false),
	"NOTICE": envBool("LOG_NOTICE", true),
	"WARN":   envBool("LOG_WARN", true),
	"ERROR":  envBool("LOG_ERROR", true),
	"FATAL":  envBool("LOG_FATAL", true),
	"STACK":  envBool("LOG_STACK", true),
}

var provenanceFormat = envString("LOG_FORMAT", SimpleFormat)
var sanityReplacements = map[string]string{}
var useFuncName = strings.Index(provenanceFormat, "%[1]s") >= 0
var logEnterStatus = envString("LOG_ENTER_STATUS", "METHOD")

func init() {
	log.SetFlags(log.Flags() | log.LUTC)
	s := envString("LOG_SANITY", "")
	if s != "" {
		var m map[string]string
		if json.Unmarshal([]byte(s), &m) == nil {
			SetSanityReplacements(m)
		} else {
			log.Printf("%-6s: %s", "WARN", "var LOG_SANITY is configured with bad value and ignored")
		}
	}
}

func UseFuncName(b bool)  { useFuncName = b }
func UsingFuncName() bool { return useFuncName }

// GetProvenanceFormat get the format used to print provenance
func GetProvenanceFormat() string {
	return provenanceFormat
}

// SetProvenanceFormat set the format used to print provenance
func SetProvenanceFormat(s string) {
	provenanceFormat = strings.TrimSpace(s)
	useFuncName = strings.Index(provenanceFormat, "%[1]s") >= 0
}

// GetSanityReplacements get the map of strings that get replaced
func GetSanityReplacements() map[string]string {
	return sanityReplacements
}

// SetSanityReplacements set the map of strings that get replaced
func SetSanityReplacements(sr map[string]string) {
	sanityReplacements = sr
}

func sanitize(s string) string {
	for k, v := range sanityReplacements {
		s = strings.Replace(s, k, v, 1)
	}
	return s
}

// FuncFileLine - fetch file/line information as a string
func FuncFileLine(depth int) (funName string, dir string, file string, line int) {
	funName, dir, file, line = na, na, na, -1
	if pc, fileName, fileLine, ok := runtime.Caller(depth); ok {
		dir = sanitize(filepath.Base(filepath.Dir(fileName)))
		file = sanitize(filepath.Base(fileName))
		line = fileLine

		if useFuncName {
			// TODO: test timing of this!
			funP := runtime.FuncForPC(pc)
			if funP != nil {
				bits := strings.Split(funP.Name(), "/")
				funName = sanitize(bits[len(bits)-1])
			}
		}
	}
	return
}

// IsLogging are we logging at this level?
func IsLogging(level string) bool {
	doit, ok := levels[level]
	return doit && ok
}

// SetLogging change logging behaviour for a level.
// Returns true iff a change was made
func SetLogging(level string, b bool) bool {
	if doit, ok := levels[level]; ok && doit != b {
		levels[level] = b
		return true
	}
	return false
}

// Logging set logging for a level
// Silently does nothing if the level is unknown.
func Logging(level string, on bool) {
	if _, ok := levels[level]; ok {
		levels[level] = on
	}
}

// Log generic log message.
func Log(level string, offset int, a ...interface{}) (x bool) {
	if doit, ok := levels[level]; !ok {
		panic("log level [" + level + "] does not exist")
	} else if doit {
		return logIt(offset+3, level, fmt.Sprint(a...))
	} else {
		return false
	}
}

func logIt(offset int, level string, text string) (x bool) {
	if provenanceFormat != "" {
		funName, dir, file, line := FuncFileLine(offset)
		provenance := fmt.Sprintf(provenanceFormat, funName, dir, file, line)
		log.Printf("%-6s: %s %s", level, strings.TrimSpace(text), provenance)
	} else {
		log.Printf("%-6s: %s", level, strings.TrimSpace(text))
	}
	return true
}

// In - used in a defer to bookend a call
// e.g. defer logger.Out(logger.In("my_func"))
// A TRACE message is generated at entry and exit from the function
func In(a ...interface{}) (now time.Time, text string) {
	text, now = fmt.Sprint(a...), time.Now()
	if text == "" {
		text, _, _, _ = FuncFileLine(2)
	}
	_ = levels[logEnterStatus] && logIt(3, logEnterStatus, ">> "+text)
	return
}

// Inf - used in a defer to bookend a call
// e.g. defer logger.Un(logger.Inf("my_func %s", arg))
func Inf(format string, a ...interface{}) (now time.Time, text string) {
	text, now = fmt.Sprintf(format, a...), time.Now()
	if text == "" {
		text, _, _, _ = FuncFileLine(2)
	}
	_ = levels["TRACE"] && logIt(3, "TRACE", ">> "+text)
	return
}

// Out - used in a defer to bookend a call
// e.g. defer logger.Out(logger.In("my_func"))
func Out(startTime time.Time, text string) {
	_ = levels[logEnterStatus] && logIt(3, logEnterStatus, fmt.Sprint("<< ", text, " ELAPSED ", time.Since(startTime)))
}

// Trace - Print a nice message
func Trace(a ...interface{}) bool {
	return levels["TRACE"] && logIt(3, "TRACE", fmt.Sprint(a...))
}

// Tracef - print a formatted message
func Tracef(format string, a ...interface{}) bool {
	return levels["TRACE"] && logIt(3, "TRACE", fmt.Sprintf(format, a...))
}

// Notice - Print a nice message
func Notice(a ...interface{}) bool {
	return levels["NOTICE"] && logIt(3, "NOTICE", fmt.Sprint(a...))
}

// Noticef - print a formatted message
func Noticef(format string, a ...interface{}) bool {
	return levels["NOTICE"] && logIt(3, "NOTICE", fmt.Sprintf(format, a...))
}

// Warn - Print a nice message
func Warn(a ...interface{}) bool {
	return levels["WARN"] && logIt(3, "WARN", fmt.Sprint(a...))
}

// Warnf - print a formatted message
func Warnf(format string, a ...interface{}) bool {
	return levels["WARN"] && logIt(3, "WARN", fmt.Sprintf(format, a...))
}

// Warnpf - print a formatted message as if from the parent frame
func Warnpf(format string, a ...interface{}) bool {
	return levels["WARN"] && logIt(4, "WARN", fmt.Sprintf(format, a...))
}

// Error - Print a nice message
func Error(a ...interface{}) bool {
	return levels["ERROR"] && logIt(3, "ERROR", fmt.Sprint(a...))
}

// Errorf - print a formatted message
func Errorf(format string, a ...interface{}) bool {
	return levels["ERROR"] && logIt(3, "ERROR", fmt.Sprintf(format, a...))
}

// Fatal - Print a nice message
func Fatal(a ...interface{}) bool {
	defer os.Exit(1)
	defer stack(4)
	return levels["FATAL"] && logIt(3, "FATAL", fmt.Sprint(a...))
}

// Fatalf - print a formatted message
func Fatalf(format string, a ...interface{}) bool {
	defer os.Exit(1)
	defer stack(4)
	return levels["FATAL"] && logIt(3, "FATAL", fmt.Sprintf(format, a...))
}

func stack(i int) {
	for ; i < 20; i++ {
		funName, dir, file, line := FuncFileLine(i)
		if dir == na {
			break
		}
		log.Println("STACK : ..", fmt.Sprintf(provenanceFormat, funName, dir, file, line))
	}
}

// Stack -- print a stack trace
func Stack(a ...interface{}) (b bool) {
	if levels["STACK"] {
		b = true
		if len(a) == 0 {
			stack(3)
		} else {
			logIt(3, "STACK", fmt.Sprint(a...))
			stack(4)
		}
	}
	return b
}

// Stackf - print a formatted message and stack trace
func Stackf(format string, a ...interface{}) (b bool) {
	if levels["STACK"] {
		b = true
		logIt(3, "STACK", fmt.Sprintf(format, a...))
		stack(4)
	}
	return
}
