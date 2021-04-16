package app

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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

type ExifData struct {
	Data        ExifToolData
	DataDump    RawJsonMap
	DataDumpRaw string
}

func GetFileMetadata(params CmdOptions, path string, info os.FileInfo) (FileMeta, error) {
	ext := strings.ToLower(filepath.Ext(path))

	fdata := FileMeta{
		Source: FilePathInfo{
			Path:      path,
			Basename:  strings.Replace(info.Name(), ext, "", -1),
			Dirname:   filepath.Dir(path),
			Extension: ext,
		},
		Size:              info.Size(),
		Checksum:          FileCalcChecksum(path),
		MediaType:         getMediaType(ext),
		IsDuplication:     false,
		IsAlreadyImported: false,
	}

	// Parse metadata
	fdata.Exif = parseMetadata(params, fdata)
	fdata.GPS = GPSDataParse(fdata.Exif.Data.GPSPosition)

	// Find file times
	fdata.ModificationTime = info.ModTime().Format(DateFormat)
	fdata.CreationTime = fdata.ModificationTime
	fdata.CreationTime = parseEarliestCreationDate(fdata)

	// Find creation tool, camera, topic
	fdata.CameraModel = parseExifCameraName(fdata.Exif.Data)
	fdata.CreationTool = parseExifCreationTool(fdata.Exif.Data)
	fdata.IsScreenShot = isScreenShot(fdata.Source.Path +
		":" + fdata.Exif.Data.SourceFile +
		":" + fdata.Exif.Data.CreatorTool +
		":" + fdata.CameraModel)

	// Build Destination file name and dirName
	fdata.Destination = buildDestination(params.DestDir, fdata)
	fdata.MetadataPath = buildChecksumPath(params.DestDir, fdata.Checksum, fdata.Source.Extension)
	alreadyExists := PathExists(fdata.Destination.Path) || PathExists(fdata.MetadataPath.Path)

	if alreadyExists {
		// Detect duplication by checksum or Destination path (e.g. when trying to copy twice from same folder)
		if filepath.Base(path) == filepath.Base(fdata.Destination.Path) {
			// skip storing duplicate if same filename
			fdata.IsAlreadyImported = true
			return fdata, nil
		}
		fdata.IsDuplication = true
		return fdata, nil
	}

	return fdata, nil
}

func buildChecksumPath(destDirRoot string, checksum string, fileExtension string) FilePathInfo {
	checksumRelDir := fmt.Sprintf("%s/%s/%s", DirMetadata, checksum[0:2], checksum[2:3])
	checksumBaseName := fmt.Sprintf("%s%s", checksum, sanitizeExtension(fileExtension))

	checksumPathInfo := FilePathInfo{
		Basename:  checksumBaseName,
		Dirname:   checksumRelDir,
		Extension: ".json",
		Path:      fmt.Sprintf("%s/%s/%s", destDirRoot, checksumRelDir, checksumBaseName) + ".json",
	}

	return checksumPathInfo
}

func buildDestination(destDirRoot string, data FileMeta) FilePathInfo {
	t, err := time.Parse(time.RFC3339, data.CreationTime)
	HandleError(err)

	ext := sanitizeExtension(data.Source.Extension)

	var dateFolder, destFilename string

	dateFolder = fmt.Sprintf("%d/%02d", t.Year(), t.Month())

	destFilename = fmt.Sprintf("%d%02d%02d-%02d%02d%02d", t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second()) + "-" + data.Checksum

	destDirName := getMediaTypeDir(data.MediaType) + "/" + dateFolder

	return FilePathInfo{
		Basename:  destFilename,
		Dirname:   destDirName,
		Extension: ext,
		Path:      destDirRoot + "/" + destDirName + "/" + destFilename + ext,
	}
}

func sanitizeExtension(ext string) string {
	ext = strings.ToLower(ext)

	switch ext {
	case "jpeg":
		return "jpg"
	}
	return ext
}

func isScreenShot(searchStr string) bool {
	if regexp.MustCompile(RegexScreenShot).MatchString(searchStr) {
		return true
	}

	return false
}

func getMediaType(ext string) string {
	if regexp.MustCompile(RegexImage).MatchString(ext) {
		return MediaTypeImage
	}

	if regexp.MustCompile(RegexVideo).MatchString(ext) {
		return MediaTypeVideo
	}

	return ""
}

func getMediaTypeDir(mediaType string) string {
	switch mediaType {
	case MediaTypeImage:
		return DirImages
	case MediaTypeVideo:
		return DirVideos
	}

	return "others"
}

func parseExifCreationTool(data ExifToolData) string {
	tool := ""

	if data.CreatorTool != "" {
		tool += data.CreatorTool
	}

	if data.Software != "" {
		tool += " " + data.Software
	}

	return strings.TrimSpace(tool)
}

func parseExifCameraName(data ExifToolData) string {
	camera := ""

	if data.Make != "" {
		camera = data.Make
	}

	if data.Model != "" {
		camera += " " + data.Model
	}

	return strings.TrimSpace(camera)
}

func parseEarliestCreationDate(data FileMeta) string {
	var dates [7][2]string
	var foundDates []time.Time
	var creationDate time.Time

	metadataDateFormat := "2006:01:02 15:04:05"

	dates[0] = [2]string{data.ModificationTime, DateFormat}
	dates[1] = [2]string{data.Exif.Data.CreateDate, metadataDateFormat}
	dates[2] = [2]string{data.Exif.Data.DateTimeOriginal, metadataDateFormat}
	dates[3] = [2]string{data.Exif.Data.DateTimeDigitized, metadataDateFormat}
	dates[4] = [2]string{data.Exif.Data.GPSDateTime, metadataDateFormat + "Z"}
	dates[5] = [2]string{data.Exif.Data.FileModifyDate, metadataDateFormat + "+07:00"}

	for _, val := range dates {
		if val[0] == "" {
			continue
		}
		t, err := time.Parse(val[1], val[0])
		if IsError(err) {
			continue
		}

		foundDates = append(foundDates, t)
	}

	for i, val := range foundDates {
		if i == 0 {
			creationDate = val
			continue
		}

		// find min valid
		if (val.Year() > 1970) && (val.Unix() < creationDate.Unix()) {
			creationDate = val
		}
	}

	if creationDate.IsZero() {
		return data.ModificationTime
	}

	return FormatDateWithTimezone(creationDate, data.GPS.Timezone)
}

func parseMetadata(params CmdOptions, fdata FileMeta) ExifData {
	metadataBytes := readExifMetadata(params, fdata)

	var metadataByteArr []RawJsonMap
	jsonerr := json.Unmarshal(metadataBytes, &metadataByteArr)

	exifData := ExifData{
		Data:        parseExifMetadata(metadataBytes),
		DataDumpRaw: string(metadataBytes),
	}

	if !IsError(jsonerr) && len(metadataByteArr) > 0 {
		exifData.DataDump = metadataByteArr[0]
	}

	return exifData
}

func parseExifMetadata(jsonData []byte) ExifToolData {
	var dataList []RawJsonMap
	HandleError(json.Unmarshal(jsonData, &dataList))
	d := dataList[0]

	ds := ExifToolData{}
	ds.SourceFile = GetJsonMapValue(d, "SourceFile")
	ds.Directory = GetJsonMapValue(d, "Directory")
	ds.FileName = GetJsonMapValue(d, "FileName")
	ds.FileSize = GetJsonMapValue(d, "FileSize")
	ds.FileModifyDate = GetJsonMapValue(d, "FileModifyDate")
	ds.FileAccessDate = GetJsonMapValue(d, "FileAccessDate")
	ds.FileType = GetJsonMapValue(d, "FileType")
	ds.FileTypeExtension = GetJsonMapValue(d, "FileTypeExtension")
	ds.FilePermissions = GetJsonMapValue(d, "FilePermissions")
	ds.MIMEType = GetJsonMapValue(d, "MIMEType")
	ds.Make = GetJsonMapValue(d, "Make")
	ds.Model = GetJsonMapValue(d, "Model")
	ds.Software = GetJsonMapValue(d, "Software")
	ds.CreatorTool = GetJsonMapValue(d, "CreatorTool")
	ds.CreateDate = GetJsonMapValue(d, "CreateDate")
	ds.ModifyDate = GetJsonMapValue(d, "ModifyDate")
	ds.DateTimeOriginal = GetJsonMapValue(d, "DateTimeOriginal")
	ds.DateTimeDigitized = GetJsonMapValue(d, "DateTimeDigitized")
	ds.ImageWidth = GetJsonMapValue(d, "ImageWidth")
	ds.ImageHeight = GetJsonMapValue(d, "ImageHeight")
	ds.ImageSize = GetJsonMapValue(d, "ImageSize")
	ds.GPSAltitude = GetJsonMapValue(d, "GPSAltitude")
	ds.GPSLatitude = GetJsonMapValue(d, "GPSLatitude")
	ds.GPSLongitude = GetJsonMapValue(d, "GPSLongitude")
	ds.GPSLatitudeRef = GetJsonMapValue(d, "GPSLatitudeRef")
	ds.GPSLongitudeRef = GetJsonMapValue(d, "GPSLongitudeRef")
	ds.GPSPosition = GetJsonMapValue(d, "GPSPosition")
	ds.GPSDateTime = GetJsonMapValue(d, "GPSDateTime")

	return ds
}

func readExifMetadata(params CmdOptions, file FileMeta) []byte {
	// Search for an already existing JSON metadata file
	pathsLookup := []string{
		// src, MD5
		buildChecksumPath(params.SrcDir, file.Checksum, file.Source.Extension).Path,
		// dest, MD5
		buildChecksumPath(params.DestDir, file.Checksum, file.Source.Extension).Path,
	}

	for _, srcMetaFile := range pathsLookup {
		if PathExists(srcMetaFile) {
			metadataBytes, err := ioutil.ReadFile(srcMetaFile)
			if !IsError(err) {
				var meta FileMeta
				jsonerr := json.Unmarshal(metadataBytes, &meta)
				HandleError(jsonerr)
				return []byte(meta.Exif.DataDumpRaw)
			}
		}
	}

	fallbackMetadata := []byte(`[{"SourceFile":"` + file.Source.Path + `", "Error": true}]`)

	jsonBytes, err := exec.Command("exiftool", file.Source.Path, "-json").Output()
	if IsError(err) {
		return fallbackMetadata
	}

	return jsonBytes
}
