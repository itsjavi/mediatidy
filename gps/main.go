package gps

import (
	"strconv"
	"strings"
)

type Position struct {
	Latitude  float64
	Longitude float64
}

// parses a string like `2 deg 38' 40.34" E`
func ParsePositionPart(val string) float64 {
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
func ParsePosition(position string) Position {
	latLng := strings.Split(strings.TrimSpace(position), ",")

	if len(latLng) != 2 {
		panic("Cannot parse GPS position: " + position)
	}

	lat := ParsePositionPart(strings.TrimSpace(latLng[0]))
	lng := ParsePositionPart(strings.TrimSpace(latLng[1]))

	return Position{lat, lng}
}
