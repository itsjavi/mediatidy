package main

import (
	"fmt"
	"strconv"
	"strings"
)

type GPSCoord struct {
	Latitude  float64
	Longitude float64
}

// parses a string like `2 deg 38' 40.34" E`
func parseGPSPositionPart(val string) float64 {
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
func parseGPSPosition(position string) GPSCoord {
	latLng := strings.Split(strings.TrimSpace(position), ",")

	if len(latLng) != 2 {
		panic("Cannot parse GPS position: " + position)
	}

	lat := parseGPSPositionPart(strings.TrimSpace(latLng[0]))
	lng := parseGPSPositionPart(strings.TrimSpace(latLng[1]))

	return GPSCoord{lat, lng}
}

func catch(e error, data ... interface{}) {
	if e != nil {
		logLn("%s\n", data)
		panic(e)
	}
}

func isError(e error) bool {
	return e != nil
}

func safeString(val interface{}) string {
	switch val.(type) {
	case int:
		return strconv.Itoa(val.(int))
	case float64:
		return strconv.FormatFloat(val.(float64), 'f', 6, 64)
	default:
		return fmt.Sprintf("%s", val)
	}
}

func logLn(message string, a ...interface{}) {
	fmt.Printf("["+AppName+"] "+message+"\n", a...)
}

func logSameLn(format string, args ... interface{}) {
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
