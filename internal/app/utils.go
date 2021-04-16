package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	tm "github.com/buger/goterm"
)

func IsError(e error) bool {
	return e != nil
}

func HandleError(e error) {
	if IsError(e) {
		log.Fatal(fmt.Sprintf("[%s] ERROR: %s\n", AppName, e))
	}
}

func PrintLn(template string, args ...interface{}) {
	fmt.Printf("["+AppName+"] "+template+"\n", args...)
}

func PrintReplaceLn(template string, args ...interface{}) {
	tm.Clear()
	tm.MoveCursor(1, 1)
	tm.Printf(template, args...)
	tm.Flush()
}

func TotalBytesToString(b int64, useDecimalSystem bool) string {
	unit := int64(1024)
	format := "%.1f %ciB"

	if useDecimalSystem == true {
		// decimal system
		unit = 1000
		format = "%.1f %cB"
	}

	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf(format, float64(b)/float64(div), "kMGTPE"[exp])
}

func ToString(val interface{}) string {
	switch val.(type) {
	case int:
		return strconv.Itoa(val.(int))
	case float64:
		return strconv.FormatFloat(val.(float64), 'f', 6, 64)
	default:
		return fmt.Sprintf("%s", val)
	}
}

func FormatDateWithTimezone(date time.Time, timezone string) string {
	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if !IsError(err) {
			date = date.In(loc)
		}
	}

	return date.Format(DateFormat)
}

func ParseDateWithTimezone(layout string, value string, timezone string) (time.Time, error) {
	t, err := time.Parse(layout, value)

	if !IsError(err) && (timezone != "") {
		loc, err := time.LoadLocation(timezone)
		if !IsError(err) {
			t = t.In(loc)
		}
	}

	return t, err
}

func GetJsonMapValue(dataMap RawJsonMap, key string) string {
	if val, ok := dataMap[key]; ok {
		return ToString(val)
	}

	return ""
}

func JsonEncodePretty(v interface{}) ([]byte, error) {
	meta, err := json.Marshal(v)

	if IsError(err) {
		return meta, err
	}

	var out bytes.Buffer
	err = json.Indent(&out, meta, "", "  ")

	return out.Bytes(), err
}
