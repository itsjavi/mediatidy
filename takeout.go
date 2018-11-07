package main

import (
	"encoding/json"
	"io/ioutil"
	"strconv"
	"time"
)

type PhotoTakenTime struct {
	Timestamp string `json:"timestamp"`
	Formatted string `json:"formatted"`
}

type GoogleTakeoutMetadata struct {
	PhotoTakenTime PhotoTakenTime `json:"PhotoTakenTime"`
}

func GetPhotoTakenTime(fileData FileData, lookupPath string) string {
	filePath := fileData.Path + ".json"

	if !pathExists(filePath) {
		filePath = checksumPath(fileData.Checksum, fileData.Extension, lookupPath) + ".takeout.json"
	}

	if !pathExists(filePath) {
		return ""
	}

	metadataBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return ""
	}

	var rawData GoogleTakeoutMetadata
	err = json.Unmarshal(metadataBytes, &rawData)

	if err != nil {
		return ""
	}

	timestamp, err := strconv.Atoi(rawData.PhotoTakenTime.Timestamp)

	if err != nil || timestamp <= 1 {
		return ""
	}

	return formatDate(time.Unix(int64(timestamp), 0), DefaultTimezone)
}
