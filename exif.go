package main

import (
	"encoding/json"
	"fmt"
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
	Duration          string
	MediaDuration     string
	TrackDuration     string
	FullJsonDump      string
}

func GetFileMetadata2() {
	//1. if in src db, use that FileMeta with that exif
	// otherwise create new FileMeta and parse exif
	//2. re-apply exif data special parsers (GPS, etc)
	//3. complete fields not saved in DB
}

func GetFileMetadata(ctx AppContext, path string, info os.FileInfo) (FileMeta, error) {
	fileExtension := strings.ToLower(filepath.Ext(path))

	fileMeta := FileMeta{
		Origin: FilePathInfo{
			Path:      path,
			Basename:  strings.Replace(info.Name(), fileExtension, "", -1),
			Dirname:   filepath.Dir(path),
			Extension: fileExtension,
		},
		Size:                        info.Size(),
		Checksum:                    FileCalcChecksum(path),
		MediaType:                   getMediaType(fileExtension),
		IsDuplicationByChecksum:     false,
		IsDuplicationByDestBasename: false,
	}

	// Parse metadata
	fileMeta.Exif = parseMetadata(ctx, fileMeta)
	gpsData := GPSDataParse(fileMeta.Exif.GPSPosition, fileMeta.Exif.GPSAltitude)
	fileMeta.GPSAltitude = gpsData.Position.Altitude
	fileMeta.GPSLatitude = ToString(gpsData.Position.Latitude)
	fileMeta.GPSLongitude = ToString(gpsData.Position.Longitude)
	fileMeta.GPSTimezone = gpsData.Timezone

	// Find file times
	fileMeta.ModificationDate = info.ModTime().Format(DateLayout)
	fileMeta.CreationDate = fileMeta.ModificationDate
	fileMeta.CreationDate = parseEarliestCreationDate(fileMeta)

	// Find creation tool, camera, topic
	fileMeta.CameraModel = parseExifCameraName(fileMeta.Exif)
	fileMeta.CreationTool = parseExifCreationTool(fileMeta.Exif)
	fileMeta.Width = fileMeta.Exif.ImageWidth
	fileMeta.Height = fileMeta.Exif.ImageHeight
	fileMeta.Duration = ParseMediaDuration(fileMeta.Exif)
	fileMeta.IsScreenShot = isScreenShot(fileMeta.Origin.Path +
		":" + fileMeta.Exif.SourceFile +
		":" + fileMeta.Exif.CreatorTool +
		":" + fileMeta.CameraModel)

	// Build Destination file name and dirName
	fileMeta.Destination = buildDestination(ctx.DestDir, fileMeta)
	alreadyExists := PathExists(fileMeta.Destination.Path) || ctx.Db.HasFileMetaByChecksum(fileMeta.Checksum)

	if alreadyExists {
		// Detect duplication by checksum or Destination path (e.g. when trying to copy twice from same folder)
		if filepath.Base(path) == filepath.Base(fileMeta.Destination.Path) {
			// skip storing duplicate if same filename
			fileMeta.IsDuplicationByDestBasename = true
			return fileMeta, nil
		}
		fileMeta.IsDuplicationByChecksum = true
		return fileMeta, nil
	}

	return fileMeta, nil
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
	t, err := time.Parse(time.RFC3339, data.CreationDate)
	HandleError(err)

	ext := sanitizeExtension(data.Origin.Extension)

	var dateFolder, destFilename string

	dateFolder = fmt.Sprintf("%d/%02d", t.Year(), t.Month())
	//dateFolder = fmt.Sprintf("%d/%02d-%02d", t.Year(), t.Month(), t.Day())

	destFilename = fmt.Sprintf("%d%02d%02d%02d%02d%02d", t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second()) + "-" + data.Checksum[0:6]

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

func ParseMediaDuration(data ExifToolData) string {
	str := ""

	if data.Duration != "" {
		return data.Duration
	}

	if data.MediaDuration != "" {
		return data.MediaDuration
	}

	if data.TrackDuration != "" {
		return data.TrackDuration
	}

	return strings.TrimSpace(str)
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

	dates[0] = [2]string{data.ModificationDate, DateLayout}
	dates[1] = [2]string{data.Exif.CreateDate, metadataDateFormat}
	dates[2] = [2]string{data.Exif.DateTimeOriginal, metadataDateFormat}
	dates[3] = [2]string{data.Exif.DateTimeDigitized, metadataDateFormat}
	dates[4] = [2]string{data.Exif.GPSDateTime, metadataDateFormat + "Z"}
	dates[5] = [2]string{data.Exif.FileModifyDate, metadataDateFormat + "+07:00"}

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
		return data.ModificationDate
	}

	return FormatDateWithTimezone(creationDate, data.GPSTimezone)
}

func parseMetadata(params AppContext, fdata FileMeta) ExifToolData {
	metadataBytes := readExifMetadata(params, fdata)

	var metadataByteArr []RawJsonMap
	jsonerr := json.Unmarshal(metadataBytes, &metadataByteArr)
	HandleError(jsonerr)

	exif := ParseExifMetadata(metadataBytes)
	exif.FullJsonDump = string(metadataBytes)

	return exif
}

func ParseExifMetadata(jsonData []byte) ExifToolData {
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
	ds.Duration = GetJsonMapValue(d, "Duration")
	ds.MediaDuration = GetJsonMapValue(d, "MediaDuration")
	ds.TrackDuration = GetJsonMapValue(d, "TrackDuration")

	return ds
}

func readExifMetadata(ctx AppContext, file FileMeta) []byte {
	// find in SRC DB
	if ctx.HasSrcMetadataDb() {
		foundFileMeta, found, err := ctx.SrcDb.FindFileMetaByChecksum(file.Checksum)
		if found {
			fmt.Print(" // exiftool data found in SRC db")
			return []byte(foundFileMeta.ExifJson)
		}
		HandleError(err)
	}
	// find in DEST DB
	if ctx.HasMetadataDb() {
		foundFileMeta, found, err := ctx.Db.FindFileMetaByChecksum(file.Checksum)
		if found {
			fmt.Print(" // exiftool data found in DEST db")
			return []byte(foundFileMeta.ExifJson)
		}
		HandleError(err)
	}

	fallbackMetadata := []byte(`[{"SourceFile":"` + file.Origin.Path + `", "Error": true}]`)

	jsonBytes, err := exec.Command("exiftool", file.Origin.Path, "-json", "-api", "largefilesupport=1", "-extractEmbedded").Output()
	if IsError(err) {
		return fallbackMetadata
	}
	fmt.Print(" // exiftool data extracted from file")

	return jsonBytes
}
