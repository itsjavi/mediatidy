package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	tm "github.com/buger/goterm"
	"github.com/ztrue/tracerr"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type KeyValueMap map[string]interface{}

func (kv KeyValueMap) GetString(key string) string {
	if val, ok := kv[key]; ok {
		return ToString(val)
	}

	return ""
}

func (kv KeyValueMap) GetInt(key string) int {
	if val, ok := kv[key]; ok {
		return StrToInt(ToString(val))
	}

	return 0
}

func IsError(e error) bool {
	return e != nil
}

func PrintLnRed(str string) {
	fmt.Println("\n" + tm.Color(str, tm.RED))
}

func PrintError(err error) {
	PrintLnRed(fmt.Sprintf("%s", err))
}

func Catch(err error) {
	if IsError(err) {
		tracerr.PrintSourceColor(tracerr.Wrap(fmt.Errorf("[%s] ERROR: %s", AppName, err)))
		log.Fatal(fmt.Sprintf("Uncaught error '%s'", err))
	}
}

func HandleErrorWithMessage(err error, msg string) {
	if IsError(err) {
		log.Fatalln(fmt.Sprintf("[%s] ERROR: %s %s", AppName, err, msg))
	}
}

func PrintLn(template string, args ...interface{}) {
	fmt.Printf("["+AppName+"] "+template+"\n", args...)
}

func PrintReplaceLn(template string, args ...interface{}) {
	fmt.Printf("\033[2K\r"+template, args...)
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

	return fmt.Sprintf(format, float64(b)/float64(div), "KMGTPE"[exp])
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

func DateInTimezone(date time.Time, timezone string) time.Time {
	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if !IsError(err) {
			date = date.In(loc)
		}
	}

	return date
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

func JsonEncodePretty(v interface{}) ([]byte, error) {
	meta, err := json.Marshal(v)

	if IsError(err) {
		return meta, err
	}

	var out bytes.Buffer
	err = json.Indent(&out, meta, "", "  ")

	return out.Bytes(), err
}

func FindEarliestDate(dates []time.Time, fallback time.Time) time.Time {
	var earliest time.Time

	// find earliest valid date
	for i, val := range dates {
		if val.Year() <= 1970 {
			continue
		}

		if i == 0 {
			earliest = val
			continue
		}

		if val.Unix() < earliest.Unix() {
			earliest = val
		}
	}

	if earliest.IsZero() {
		return fallback
	}

	return earliest
}

func StrToInt(str string) int {
	if str == "" {
		return 0
	}
	num, err := strconv.Atoi(strings.Split(strings.Replace(str, ",", ".", -1), ".")[0])
	if IsError(err) {
		Catch(err)
	}
	return num
}

func NormalizeTimestampStringFormat(date string) (string, string) {
	layout := "2006-01-02T15:04:05"
	date = strings.ToUpper(date)
	date = strings.Replace(date, " ", "T", 1)
	date = strings.ReplaceAll(date, "GMT", "")
	date = strings.Trim(date, " Z")

	withSemicolonYmd := regexp.MustCompile("(?i)^([0-9]{4}):([0-9]{2}):([0-9]{2})(.*)")
	withoutSeconds := regexp.MustCompile("(?i)^([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2})([^:].*$|$)")
	withLeapSeconds := regexp.MustCompile("(?i)^([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2})([.:])([0-9]{2,})(.*)?")
	withTimeZone := regexp.MustCompile("(?i)(.*)([+\\-Z])([0-9]{2}:[0-9]{2})$")

	if withSemicolonYmd.MatchString(date) { // replace : with - in date
		date = withSemicolonYmd.ReplaceAllString(date, "$1-$2-$3$4")
	}

	if withoutSeconds.MatchString(date) { // add seconds part to date
		date = withoutSeconds.ReplaceAllString(date, "$1:00$2")
	}

	if withLeapSeconds.MatchString(date) { // add leap seconds to layout
		layout += withLeapSeconds.ReplaceAllString(date, "$2") +
			strings.Repeat("0", len(withLeapSeconds.ReplaceAllString(date, "$3")))
	}

	if withTimeZone.MatchString(date) { // add timezone to layout
		layout += "Z07:00"
	}

	return layout, date
}

func MaxInt(vars ...int) int {
	max := vars[0]

	for _, n := range vars {
		if n > max {
			max = n
		}
	}

	return max
}

func MinInt(vars ...int) int {
	min := vars[0]

	for _, n := range vars {
		if n < min {
			min = n
		}
	}

	return min
}
