package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type ExifToolData struct {
	// File info:
	SourceFile        string
	Directory         string
	FileName          string
	FileSize          string
	FileModifyDate    string
	FileAccessDate    string
	FileType          string
	FileTypeExtension string
	FilePermissions   string
	MIMEType          string
	// Exif / XMP info:
	Make              string
	Model             string
	Software          string
	CreatorTool       string
	CreateDate        string
	ModifyDate        string
	DateTimeOriginal  string
	DateTimeDigitized string
	ImageWidth        string
	ImageHeight       string
	ImageSize         string
	GPSAltitude       string
	GPSLatitude       string
	GPSLongitude      string
}

type ExifToolDataMap map[string]interface{}

func extractMetadata(path string, fallback []byte) []byte {
	jsonStr, err := exec.Command("exiftool", path, "-json").Output()
	if err != nil {
		return fallback
	}

	return jsonStr
}

func unmarshalMetadata(jsonData []byte) ExifToolData {
	var dataList []ExifToolDataMap
	catch(json.Unmarshal(jsonData, &dataList), string(jsonData))
	d := dataList[0]

	ds := ExifToolData{}
	ds.SourceFile = getSanitizedValue(d, "SourceFile")
	ds.Directory = getSanitizedValue(d, "Directory")
	ds.FileName = getSanitizedValue(d, "FileName")
	ds.FileSize = getSanitizedValue(d, "FileSize")
	ds.FileModifyDate = getSanitizedValue(d, "FileModifyDate")
	ds.FileAccessDate = getSanitizedValue(d, "FileAccessDate")
	ds.FileType = getSanitizedValue(d, "FileType")
	ds.FileTypeExtension = getSanitizedValue(d, "FileTypeExtension")
	ds.FilePermissions = getSanitizedValue(d, "FilePermissions")
	ds.MIMEType = getSanitizedValue(d, "MIMEType")
	ds.Make = getSanitizedValue(d, "Make")
	ds.Model = getSanitizedValue(d, "Model")
	ds.Software = getSanitizedValue(d, "Software")
	ds.CreatorTool = getSanitizedValue(d, "CreatorTool")
	ds.CreateDate = getSanitizedValue(d, "CreateDate")
	ds.ModifyDate = getSanitizedValue(d, "ModifyDate")
	ds.DateTimeOriginal = getSanitizedValue(d, "DateTimeOriginal")
	ds.DateTimeDigitized = getSanitizedValue(d, "DateTimeDigitized")
	ds.ImageWidth = getSanitizedValue(d, "ImageWidth")
	ds.ImageHeight = getSanitizedValue(d, "ImageHeight")
	ds.ImageSize = getSanitizedValue(d, "ImageSize")
	ds.GPSAltitude = getSanitizedValue(d, "GPSAltitude")
	ds.GPSLatitude = getSanitizedValue(d, "GPSLatitude")
	ds.GPSLongitude = getSanitizedValue(d, "GPSLongitude")

	return ds
}

func getSanitizedValue(dataMap ExifToolDataMap, key string) string {
	if val, ok := dataMap[key]; ok {
		return sanitizeType(val)
	}

	return ""
}

func sanitizeType(val interface{}) string {
	switch val.(type) {
	case int:
		return strconv.Itoa(val.(int))
	case float64:
		return strconv.FormatFloat(val.(float64), 'f', 6, 64)
	default:
		return fmt.Sprintf("%s", val)
	}
}
