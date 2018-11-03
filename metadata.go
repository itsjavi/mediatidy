package main

import (
	"encoding/json"
	"fmt"
	"github.com/gosimple/slug"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type RawJsonMap map[string]interface{}

type FileFlags struct {
	unique     bool
	duplicated bool
	skipped    bool
}

type DestFile struct {
	path      string
	name      string
	dirName   string
	extension string
}

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
	GPSDateTime       string
}

type FileData struct {
	path             string
	relPath          string
	dir              string
	relDir           string
	metaDir          string
	duplDir          string
	dest             DestFile
	name             string
	nameSlug         string
	size             int64
	extension        string
	creationTime     string
	modificationTime string
	mediaType        string
	isMultimedia     bool
	flags            FileFlags
	metadata         ExifToolData
	metadataRaw      string
	creationTool     string
	cameraModel      string
	topic            string
	checksum         string
}

type PhotoTakenTime struct {
	Timestamp string `json:"timestamp"`
	Formatted string `json:"formatted"`
}

type GoogleTakeoutData struct {
	PhotoTakenTime PhotoTakenTime `json:"PhotoTakenTime"`
}

func readExifMetadata(path string, fallback []byte) []byte {
	// logLn("reading exif for %s", path)
	jsonBytes, err := exec.Command("exiftool", path, "-json").Output()
	if err != nil {
		return fallback
	}

	return jsonBytes
}

func parseExifMetadata(jsonData []byte) ExifToolData {
	var dataList []RawJsonMap
	catch(json.Unmarshal(jsonData, &dataList), string(jsonData))
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
	ds.GPSDateTime = getJsonMapValue(d, "GPSDateTime")

	return ds
}

func getJsonMapValue(dataMap RawJsonMap, key string) string {
	if val, ok := dataMap[key]; ok {
		return safeString(val)
	}

	return ""
}

func getMediaType(ext string) string {
	if imageReg.MatchString(ext) {
		return MediaTypeImages
	}

	if videoReg.MatchString(ext) {
		return MediaTypeVideos
	}

	if audioReg.MatchString(ext) {
		return MediaTypeAudios
	}

	if docsReg.MatchString(ext) {
		return MediaTypeDocuments
	}

	return ""
}

func readMetadata(params appParams, file FileData) []byte {
	// Search for an already existing app JSON metadata file in the Src path
	fileRelDir := filepath.Dir(strings.Replace(file.path, params.Src+"/", "", -1))
	srcMetaFile := params.Src + "/" + DirMetadata + "/" + fileRelDir + "/" +
		file.name + file.extension + ".json"

	if !pathExists(srcMetaFile) {
		// try with filename.json (old format)
		srcMetaFile = params.Src + "/" + DirMetadata + "/" + fileRelDir + "/" + file.name + ".json"
	}

	if !pathExists(srcMetaFile) {
		// try with Dest dir
		srcMetaFile = params.Dest + "/" + DirMetadata + "/" + fileRelDir + "/" +
			file.name + file.extension + ".json"
	}

	if !pathExists(srcMetaFile) {
		// try with Dest dir (old format)
		srcMetaFile = params.Dest + "/" + DirMetadata + "/" + fileRelDir + "/" + file.name + ".json"
	}

	if pathExists(srcMetaFile) {
		metadataBytes, err := ioutil.ReadFile(srcMetaFile)
		if err == nil {
			return metadataBytes
		}
	}

	fallbackMetadata := []byte(`[{"SourceFile":"` + file.path + `", "Error": true}]`)
	return readExifMetadata(file.path, fallbackMetadata)
}

func buildFileData(params appParams, path string, info os.FileInfo) (FileData, error) {
	data := FileData{path: path, size: info.Size(), flags: FileFlags{skipped: false, duplicated: false, unique: false}}

	// Unreadable path or is a dirName?
	if data.size == 0 || info.IsDir() {
		data.flags.skipped = true
		return data, nil
	}

	data.extension = strings.ToLower(filepath.Ext(path))
	data.mediaType = getMediaType(data.extension)
	data.isMultimedia = false

	// Unsupported media type?
	if data.mediaType == "" {
		data.flags.skipped = true
		return data, nil
	}

	// File extension not whitelisted?
	if data.mediaType == "" ||
		(*params.Extensions != "" &&
			!regexp.MustCompile("(?i)\\.(" + *params.Extensions + ")$").MatchString(path)) {
		data.flags.skipped = true
		return data, nil
	}

	if data.mediaType != MediaTypeDocuments {
		data.isMultimedia = true
	}

	// File is too small?
	minFileSize := FileSizeMin
	if !data.isMultimedia {
		minFileSize = FileSizeMinDocs
	}
	if data.size < int64(minFileSize) {
		data.flags.skipped = true
		return data, nil
	}

	data.dir = filepath.Dir(path)
	data.relDir = strings.Replace(data.dir, params.Src+"/", "", -1)
	data.relPath = strings.Replace(path, params.Src+"/", "", -1)
	data.metaDir = params.Src + "/" + DirMetadata + "/" + data.relDir
	data.duplDir = params.Src + "/" + DirDuplicates + "/" + data.relDir

	// Name without extension
	data.name = strings.Replace(info.Name(), data.extension, "", -1)
	data.nameSlug = slug.Make(data.name)

	// Parse metadata
	metadataBytes := readMetadata(params, data)
	data.metadata = parseExifMetadata(metadataBytes)
	data.metadataRaw = string(metadataBytes)

	// Find file times
	data.modificationTime = info.ModTime().Format(time.RFC3339)
	data.creationTime = data.modificationTime
	data.creationTime = findEarliestCreationDate(data)

	data.flags.unique = true

	// MD5 checksum
	checksum, err := fileChecksum(data.path)
	if err != nil {
		data.flags.skipped = true
		return data, err
	}

	data.checksum = checksum

	// Find creation tool, camera, topic
	data.topic = findFileTopic(path)
	data.cameraModel = findCameraName(data)
	data.creationTool = findCreationTool(data)

	// Build Dest file name and dirName
	destDir, destName := buildDestPaths(data)
	data.dest.name = destName
	data.dest.dirName = destDir
	data.dest.extension = data.extension
	data.dest.path = params.Dest + "/" + data.dest.dirName + "/" + data.dest.name + data.dest.extension

	if pathExists(checksumPath(data.mediaType, checksum, params.Dest)) || pathExists(data.dest.path) {
		// Detect duplication by checksum or Dest path
		if filepath.Base(path) == filepath.Base(data.dest.path) {
			// skip storing duplicate if same filename
			data.flags.skipped = true
		}
		data.flags.unique = false
		data.flags.duplicated = true
		return data, nil
	}

	return data, nil
}

func checksumPath(mediaType, checksum, rootPath string) string {
	checksumRelPath := fmt.Sprintf("%s/%s/%s.txt", checksum[0:2], checksum[2:3], checksum)
	return fmt.Sprintf("%s/%s/%s/%s", rootPath, DirChecksums, mediaType, checksumRelPath)
}

func buildDestPaths(data FileData) (string, string) {
	t, err := time.Parse(time.RFC3339, data.creationTime)
	if err != nil {
		panic(err)
	}

	dateFolder := "2000/01/01"
	fileNamePrefix := "20000101-000000"
	topic := data.topic

	if data.cameraModel != "" {
		topic = slug.Make(data.cameraModel)
	}

	if data.mediaType != MediaTypeImages && topic == DefaultCameraModelFallback {
		dateFolder = fmt.Sprintf("%d/%02d", t.Year(), t.Month())
	} else {
		dateFolder = fmt.Sprintf("%d/%s/%02d", t.Year(), topic, t.Month())
	}

	if data.isMultimedia {
		fileNamePrefix = fmt.Sprintf("%d%02d%02d-%02d%02d%02d", t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second())
	} else {
		fileNamePrefix = data.nameSlug
	}

	if strings.Contains(strings.ToUpper(fileNamePrefix), strings.ToUpper(data.checksum)) {
		// Avoid repeating MD5 hash in the filename
		return data.mediaType + "/" + dateFolder, fileNamePrefix
	}

	return data.mediaType + "/" + dateFolder, fileNamePrefix + "-" + data.checksum
}

func findFileTopic(path string) string {
	topic := DefaultCameraModelFallback
	tools := map[string]string{
		"screenshots": `(?i)(Screen Shot|Screenshot|Captura)`,
		"facebook":    `(?i)(facebook)`,
		"instagram":   `(?i)(instagram)`,
		"twitter":     `(?i)(twitter)`,
		"whatsapp":    `(?i)(whatsapp)`,
		"telegram":    `(?i)(telegram)`,
		"messenger":   `(?i)(messenger)`,
		"snapchat":    `(?i)(snapchat)`,
		"music":       `(?i)(music|itunes|songs|lyrics|singing|karaoke)`,
		"pokemon":     `(?i)(pokemon|poke|pkm)`,
	}

	for toolName, toolRegex := range tools {
		if regexp.MustCompile(toolRegex).MatchString(path) {
			return toolName
		}
	}

	return topic
}

func findCreationTool(data FileData) string {
	tool := ""

	if data.metadata.CreatorTool != "" {
		tool += data.metadata.CreatorTool
	}

	if tool == "" && (data.metadata.Software != "") {
		tool += data.metadata.Software
	}

	return strings.TrimSpace(tool)
}

func findCameraName(data FileData) string {
	camera := ""

	if data.metadata.Make != "" {
		camera = data.metadata.Make
	}

	if data.metadata.Model != "" {
		camera += " " + data.metadata.Model
	}

	return strings.TrimSpace(camera)
}

func findEarliestCreationDate(data FileData) string {
	var dates [6][2]string
	var foundDates []time.Time
	var creationDate time.Time

	metadataDateFormat := "2006:01:02 15:04:05"

	dates[0] = [2]string{findGoogleTakeoutTakenTimestamp(data), time.RFC3339}
	dates[1] = [2]string{data.modificationTime, time.RFC3339}
	dates[2] = [2]string{data.metadata.CreateDate, metadataDateFormat}
	dates[3] = [2]string{data.metadata.DateTimeOriginal, metadataDateFormat}
	dates[4] = [2]string{data.metadata.DateTimeDigitized, metadataDateFormat}
	dates[4] = [2]string{data.metadata.GPSDateTime, metadataDateFormat + "Z"}
	dates[5] = [2]string{data.metadata.FileModifyDate, metadataDateFormat + "+07:00"}

	for _, val := range dates {
		if val[0] == "" {
			continue
		}
		t, err := time.Parse(val[1], val[0])
		if err != nil {
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
		return data.modificationTime
	}

	return creationDate.Format(time.RFC3339)
}

func findGoogleTakeoutTakenTimestamp(fileData FileData) string {
	filePath := fileData.path + ".json"

	if !pathExists(filePath) {
		filePath = fileData.metaDir + "/" + fileData.name + fileData.extension + ".takeout.json"
	}

	if !pathExists(filePath) {
		return ""
	}

	metadataBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return ""
	}

	var rawData GoogleTakeoutData
	err = json.Unmarshal(metadataBytes, &rawData)

	if err != nil {
		return ""
	}

	timestamp, err := strconv.Atoi(rawData.PhotoTakenTime.Timestamp)

	if err != nil || timestamp <= 1 {
		return ""
	}

	return time.Unix(int64(timestamp), 0).Format(time.RFC3339)
}
