package main

import (
	"encoding/json"
	"os/exec"
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

func extractMetadata(path string, fallback []byte) []byte {
	jsonStr, err := exec.Command("exiftool", path, "-json").Output()
	if err != nil {
		return fallback
	}

	return jsonStr
}

func unmarshalMetadata(jsonData []byte) ExifToolData {
	var dataList []RawJsonMap
	catch(json.Unmarshal(jsonData, &dataList), string(jsonData))
	d := dataList[0]

	ds := ExifToolData{}
	ds.SourceFile = getMapValueByKey(d, "SourceFile")
	ds.Directory = getMapValueByKey(d, "Directory")
	ds.FileName = getMapValueByKey(d, "FileName")
	ds.FileSize = getMapValueByKey(d, "FileSize")
	ds.FileModifyDate = getMapValueByKey(d, "FileModifyDate")
	ds.FileAccessDate = getMapValueByKey(d, "FileAccessDate")
	ds.FileType = getMapValueByKey(d, "FileType")
	ds.FileTypeExtension = getMapValueByKey(d, "FileTypeExtension")
	ds.FilePermissions = getMapValueByKey(d, "FilePermissions")
	ds.MIMEType = getMapValueByKey(d, "MIMEType")
	ds.Make = getMapValueByKey(d, "Make")
	ds.Model = getMapValueByKey(d, "Model")
	ds.Software = getMapValueByKey(d, "Software")
	ds.CreatorTool = getMapValueByKey(d, "CreatorTool")
	ds.CreateDate = getMapValueByKey(d, "CreateDate")
	ds.ModifyDate = getMapValueByKey(d, "ModifyDate")
	ds.DateTimeOriginal = getMapValueByKey(d, "DateTimeOriginal")
	ds.DateTimeDigitized = getMapValueByKey(d, "DateTimeDigitized")
	ds.ImageWidth = getMapValueByKey(d, "ImageWidth")
	ds.ImageHeight = getMapValueByKey(d, "ImageHeight")
	ds.ImageSize = getMapValueByKey(d, "ImageSize")
	ds.GPSAltitude = getMapValueByKey(d, "GPSAltitude")
	ds.GPSLatitude = getMapValueByKey(d, "GPSLatitude")
	ds.GPSLongitude = getMapValueByKey(d, "GPSLongitude")

	return ds
}
