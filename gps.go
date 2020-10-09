package main

import (
	"github.com/bradfitz/latlong"
	"strconv"
	"strings"
)

type GPSData struct {
	Position  string
	Altitude  string
	Latitude  float64
	Longitude float64
	DateTime  string
	Timezone  string
}

func (data *GPSData) Parse(gpsPosition string, gpsAltitude string, gpsDateTime string) {
	if gpsAltitude == "" {
		data.Altitude = "0 m Above Sea Level"
	}

	if gpsPosition == "" {
		return
	}

	coords := gpsParseCoords(gpsPosition)

	data.Position = gpsPosition
	data.Altitude = gpsAltitude
	data.Latitude = coords[0]
	data.Longitude = coords[1]
	data.DateTime = gpsDateTime
	data.Timezone = latlong.LookupZoneName(data.Latitude, data.Longitude)
}

// parses a string like `39 deg 34' 4.66" N, 2 deg 38' 40.34" E`
func gpsParseCoords(gpsPosition string) [2]float64 {
	latLng := strings.Split(strings.TrimSpace(gpsPosition), ",")

	if len(latLng) != 2 {
		panic("Cannot parse GPS position: " + gpsPosition)
	}

	return [2]float64{
		gpsParsePart(strings.TrimSpace(latLng[0])), //lat
		gpsParsePart(strings.TrimSpace(latLng[1])), //lng
	}
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
