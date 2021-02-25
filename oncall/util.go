package oncall

import (
	"fmt"
	"log"
	"strings"
)

func leveledLog(level string) func(format string, v ...interface{}) {
	prefix := fmt.Sprintf("[%s] ", strings.ToUpper(level))
	return func(format string, v ...interface{}) {
		log.Printf(prefix+format, v...)
	}
}

var traceLog = leveledLog("trace")
var debugLog = leveledLog("debug")
var infoLog = leveledLog("info")
var warnLog = leveledLog("warn")
var errorLog = leveledLog("error")
