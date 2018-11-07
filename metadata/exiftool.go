package metadata

import (
	"encoding/json"
	"github.com/itsjavi/happytimes/utils"
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
	GPSLatitudeRef    string
	GPSLongitudeRef   string
	GPSPosition       string
	GPSDateTime       string
}

func readExifMetadata(path string, fallback []byte) []byte {
	jsonBytes, err := exec.Command("exiftool", path, "-json").Output()
	if utils.IsError(err) {
		return fallback
	}

	return jsonBytes
}

func parseExifMetadata(jsonData []byte) ExifToolData {
	var dataList []RawJsonMap
	utils.Catch(json.Unmarshal(jsonData, &dataList), string(jsonData))
	d := dataList[0]

	ds := ExifToolData{}
	ds.SourceFile = getJsonMapValue(d, "SourceFile")
	ds.Directory = getJsonMapValue(d, "Directory")
	ds.FileName = getJsonMapValue(d, "FileName")
	ds.FileSize = getJsonMapValue(d, "FileSize")
	ds.FileModifyDate = getJsonMapValue(d, "FileModifyDate")
	ds.FileAccessDate = getJsonMapValue(d, "FileAccessDate")
	ds.FileType = getJsonMapValue(d, "FileType")
	ds.FileTypeExtension = getJsonMapValue(d, "FileTypeExtension")
	ds.FilePermissions = getJsonMapValue(d, "FilePermissions")
	ds.MIMEType = getJsonMapValue(d, "MIMEType")
	ds.Make = getJsonMapValue(d, "Make")
	ds.Model = getJsonMapValue(d, "Model")
	ds.Software = getJsonMapValue(d, "Software")
	ds.CreatorTool = getJsonMapValue(d, "CreatorTool")
	ds.CreateDate = getJsonMapValue(d, "CreateDate")
	ds.ModifyDate = getJsonMapValue(d, "ModifyDate")
	ds.DateTimeOriginal = getJsonMapValue(d, "DateTimeOriginal")
	ds.DateTimeDigitized = getJsonMapValue(d, "DateTimeDigitized")
	ds.ImageWidth = getJsonMapValue(d, "ImageWidth")
	ds.ImageHeight = getJsonMapValue(d, "ImageHeight")
	ds.ImageSize = getJsonMapValue(d, "ImageSize")
	ds.GPSAltitude = getJsonMapValue(d, "GPSAltitude")
	ds.GPSLatitude = getJsonMapValue(d, "GPSLatitude")
	ds.GPSLongitude = getJsonMapValue(d, "GPSLongitude")
	ds.GPSLatitudeRef = getJsonMapValue(d, "GPSLatitudeRef")
	ds.GPSLongitudeRef = getJsonMapValue(d, "GPSLongitudeRef")
	ds.GPSPosition = getJsonMapValue(d, "GPSPosition")
	ds.GPSDateTime = getJsonMapValue(d, "GPSDateTime")

	return ds
}

func getJsonMapValue(dataMap RawJsonMap, key string) string {
	if val, ok := dataMap[key]; ok {
		return utils.ToString(val)
	}

	return ""
}
