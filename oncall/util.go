package oncall

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

func leveledLog(level string) func(format string, v ...interface{}) {
	prefix := fmt.Sprintf("[%s] ", strings.ToUpper(level))
	return func(format string, v ...interface{}) {
		log.Printf(prefix+format, v...)
	}
}

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

var traceLog = leveledLog("trace")
var debugLog = leveledLog("debug")
var infoLog = leveledLog("info")
var warnLog = leveledLog("warn")
var errorLog = leveledLog("error")
