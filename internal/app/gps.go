package app

import (
	"github.com/bradfitz/latlong"
	"strconv"
	"strings"
)

type GPSCoord struct {
	Latitude  float64
	Longitude float64
}

type GPSData struct {
	Position GPSCoord
	Timezone string
}

func GPSDataParse(gpsPosition string) GPSData {
	if gpsPosition == "" {
		return GPSData{Timezone: DefaultTimezone}
	}
	data := GPSData{Position: gpsParseCoords(gpsPosition)}
	data.Timezone = latlong.LookupZoneName(data.Position.Latitude, data.Position.Longitude)

	return data
}

// parses a string like `39 deg 34' 4.66" N, 2 deg 38' 40.34" E`
func gpsParseCoords(position string) GPSCoord {
	latLng := strings.Split(strings.TrimSpace(position), ",")

	if len(latLng) != 2 {
		panic("Cannot parse GPS position: " + position)
	}

	lat := gpsParsePart(strings.TrimSpace(latLng[0]))
	lng := gpsParsePart(strings.TrimSpace(latLng[1]))

	return GPSCoord{lat, lng}
}

// parses a string like `2 deg 38' 40.34" E`
func gpsParsePart(val string) float64 {
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
