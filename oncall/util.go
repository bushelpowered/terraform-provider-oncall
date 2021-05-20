package oncall

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bushelpowered/oncall-client-go/oncall"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

func diagFromErrf(err error, fmtString string, values ...interface{}) diag.Diagnostics {
	if err == nil {
		return nil
	}
	return diag.FromErr(errors.Wrapf(err, fmtString, values...))
}

func getResourceStringSet(d *schema.ResourceData, fieldName string) []string {
	stringSet := d.Get(fieldName).(*schema.Set).List()
	stringList := make([]string, 0, len(stringSet))
	for _, s := range stringSet {
		stringList = append(stringList, s.(string))
	}
	return stringList
}

func setResourceStringSet(d *schema.ResourceData, fieldName string, values []string) {
	valSet := &schema.Set{
		F: schema.HashString,
	}
	for _, v := range values {
		valSet.Add(v)
	}
	d.Set(fieldName, valSet)
}

func stringSliceContains(slice []string, search string) bool {
	for _, s := range slice {
		if s == search {
			return true
		}
	}
	return false
}

func validateStringSliceContains(slice []string) func(interface{}, cty.Path) diag.Diagnostics {
	return func(val interface{}, path cty.Path) diag.Diagnostics {
		if !stringSliceContains(slice, val.(string)) {
			return diag.Errorf("Must be one of %v", slice)
		}
		return nil
	}
}

var traceLog = DefaultLogger{}.Trace
var debugLog = DefaultLogger{}.Debug
var infoLog = DefaultLogger{}.Info
var warnLog = DefaultLogger{}.Warn
var errorLog = DefaultLogger{}.Error

type DefaultLogger struct {
	fields map[string]interface{}
}

func (l DefaultLogger) leveledLog(level string, values ...interface{}) {
	prefix := fmt.Sprintf("[%s] Oncall Provider: %+v ", strings.ToUpper(level), l.fields)
	printThis := []interface{}{
		prefix,
	}
	printThis = append(printThis, values)
	fmt.Fprintln(os.Stderr, printThis...)
}

func (l DefaultLogger) leveledLogf(level string, format string, values ...interface{}) {
	prefix := fmt.Sprintf("[%s] Oncall Provider: %+v", strings.ToUpper(level), l.fields)
	fmt.Fprintf(os.Stderr, prefix+format+"\n", values...)
}

func (l DefaultLogger) WithField(key string, value interface{}) oncall.LeveledLogger {
	if l.fields == nil {
		l.fields = make(map[string]interface{})
	}
	l.fields[key] = value
	return l
}

func (l DefaultLogger) Trace(a ...interface{}) {
	l.leveledLog("Trace", a...)
}
func (l DefaultLogger) Tracef(format string, values ...interface{}) {
	l.leveledLogf("Trace", format, values...)
}

func (l DefaultLogger) Debug(a ...interface{}) {
	l.leveledLog("Debug", a...)
}
func (l DefaultLogger) Debugf(format string, values ...interface{}) {
	l.leveledLogf("Debug", format, values...)
}

func (l DefaultLogger) Info(a ...interface{}) {
	l.leveledLog("Info", a...)
}
func (l DefaultLogger) Infof(format string, values ...interface{}) {
	l.leveledLogf("Info", format, values...)
}

func (l DefaultLogger) Warn(a ...interface{}) {
	l.leveledLog("Warn", a...)
}
func (l DefaultLogger) Warnf(format string, values ...interface{}) {
	l.leveledLogf("Warn", format, values...)
}

func (l DefaultLogger) Error(a ...interface{}) {
	l.leveledLog("Error", a...)
}
func (l DefaultLogger) Errorf(format string, values ...interface{}) {
	l.leveledLogf("Error", format, values...)
}

func (l DefaultLogger) Fatal(a ...interface{}) {
	l.leveledLog("Fatal", a...)
	log.Fatal("Above error was fatal")
}
func (l DefaultLogger) Fatalf(format string, values ...interface{}) {
	l.leveledLogf("Fatal", format, values...)
	log.Fatal("Above error was fatal")
}
