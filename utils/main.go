package utils

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	IsUnix = runtime.GOOS == "linux" || runtime.GOOS == "darwin" ||
		runtime.GOOS == "android" || runtime.GOOS == "freebsd" || runtime.GOOS == "openbsd"
	DefaultDateFormat = time.RFC3339
	DefaultTimezone   = "Europe/Berlin"
)

func DateFormat(date time.Time, timezone string) string {
	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if !IsError(err) {
			date = date.In(loc)
		}
	}

	return date.Format(DefaultDateFormat)
}

func DateParse(layout string, value string, timezone string) (time.Time, error) {
	t, err := time.Parse(layout, value)

	if !IsError(err) && (timezone != "") {
		loc, err := time.LoadLocation(timezone)
		if !IsError(err) {
			t = t.In(loc)
		}
	}

	return t, err
}

type GPSPosition struct {
	Latitude  float64
	Longitude float64
}

// parses a string like `2 deg 38' 40.34" E`
func ParseGPSPositionPart(val string) float64 {
	chunks := strings.Split(val, " ")

	if len(chunks) != 5 {
		panic("Cannot parse GPS position: " + val)
	}

	deg, _ := strconv.ParseFloat(strings.Trim(chunks[0], " '\""), 64)
	minutes, _ := strconv.ParseFloat(strings.Trim(chunks[2], " '\""), 64)
	seconds, _ := strconv.ParseFloat(strings.Trim(chunks[3], " '\""), 64)

	coord := deg + (minutes / 60) + (seconds / 3600)
	ref := strings.ToUpper(chunks[4])

	if (ref == "S") || (ref == "W") { // N is "+", S is "-",  E is "+", W is "-"
		coord *= -1
	}

	return coord
}

// parses a string like `39 deg 34' 4.66" N, 2 deg 38' 40.34" E`
func ParseGPSPosition(position string) GPSPosition {
	latLng := strings.Split(strings.TrimSpace(position), ",")

	if len(latLng) != 2 {
		panic("Cannot parse GPS position: " + position)
	}

	lat := ParseGPSPositionPart(strings.TrimSpace(latLng[0]))
	lng := ParseGPSPositionPart(strings.TrimSpace(latLng[1]))

	return GPSPosition{lat, lng}
}

func Catch(e error, data ... interface{}) {
	if e != nil {
		LogLn("%s\n", data)
		panic(e)
	}
}

func IsError(e error) bool {
	return e != nil
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

func LogLn(message string, a ...interface{}) {
	fmt.Printf("["+AppName+"] "+message+"\n", a...)
}

func LogSameLn(format string, args ... interface{}) {
	fmt.Printf("\033[2K\r"+format, args...)
}

func ByteCountToHumanReadable(b int64, useDecimalSystem bool) string {
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
