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
	Path      string
	Name      string
	DirName   string
	Extension string
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
	Path             string
	Dir              string
	RelativeDir      string
	Name             string
	NameSlug         string
	Size             int64
	Extension        string
	CreationTime     string
	ModificationTime string
	MediaType        string
	Metadata         ExifToolData
	MetadataDump     RawJsonMap
	MetadataDumpRaw  string
	CreationTool     string
	CameraModel      string
	Topic            string
	Checksum         string
	// private:
	relativePath string
	dest         DestFile
	isMultimedia bool
	flags        FileFlags
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
	// Search for an already existing happybox-generated JSON metadata file (v2)
	pathsLookup := []string{
		// src, MD5
		checksumPath(file.Checksum, file.Extension, params.Src) + ".json",
		// dest, MD5
		checksumPath(file.Checksum, file.Extension, params.Dest) + ".json",
	}

	for _, srcMetaFile := range pathsLookup {
		if pathExists(srcMetaFile) {
			metadataBytes, err := ioutil.ReadFile(srcMetaFile)
			if err == nil {
				var meta FileData;
				jsonerr := json.Unmarshal(metadataBytes, &meta)
				if jsonerr == nil {
					return []byte(meta.MetadataDumpRaw)
				} else {
					panic(jsonerr)
				}
			}
		}
	}
	// Search for an already existing happybox-generated JSON metadata file (v1 metadata file)
	pathsLookup = []string{
		// src, filename + ext
		params.Src + "/" + DirApp + "/" + file.RelativeDir + "/" + file.Name + file.Extension + ".json",
		// dest, filename + ext
		params.Dest + "/" + DirApp + "/" + file.RelativeDir + "/" + file.Name + file.Extension + ".json",
		// src, filename
		params.Src + "/" + DirApp + "/" + file.RelativeDir + "/" + file.Name + ".json",
		// dest, filename
		params.Dest + "/" + DirApp + "/" + file.RelativeDir + "/" + file.Name + ".json",
	}

	for _, srcMetaFile := range pathsLookup {
		if pathExists(srcMetaFile) {
			metadataBytes, err := ioutil.ReadFile(srcMetaFile)
			if err == nil {
				return metadataBytes
			}
		}
	}

	fallbackMetadata := []byte(`[{"SourceFile":"` + file.Path + `", "Error": true}]`)
	return readExifMetadata(file.Path, fallbackMetadata)
}

func buildFileData(params appParams, path string, info os.FileInfo) (FileData, error) {
	data := FileData{Path: path, Size: info.Size(), flags: FileFlags{skipped: false, duplicated: false, unique: false}}

	// Unreadable path or is a dirName?
	if data.Size == 0 || info.IsDir() {
		data.flags.skipped = true
		return data, nil
	}

	data.Extension = strings.ToLower(filepath.Ext(path))
	data.MediaType = getMediaType(data.Extension)
	data.isMultimedia = false

	// Unsupported media type?
	if data.MediaType == "" {
		data.flags.skipped = true
		return data, nil
	}

	// File extension not whitelisted?
	if data.MediaType == "" ||
		(*params.Extensions != "" &&
			!regexp.MustCompile("(?i)\\.(" + *params.Extensions + ")$").MatchString(path)) {
		data.flags.skipped = true
		return data, nil
	}

	if data.MediaType != MediaTypeDocuments {
		data.isMultimedia = true
	}

	// File is too small?
	minFileSize := FileSizeMin
	if !data.isMultimedia {
		minFileSize = FileSizeMinDocs
	}
	if data.Size < int64(minFileSize) {
		data.flags.skipped = true
		return data, nil
	}

	// MD5 checksum
	checksum, err := fileChecksum(data.Path)
	if err != nil {
		data.flags.skipped = true
		return data, err
	}

	data.Checksum = checksum
	data.flags.unique = true

	data.Dir = filepath.Dir(path)
	data.RelativeDir = strings.Replace(data.Dir, params.Src+"/", "", -1)
	data.relativePath = strings.Replace(path, params.Src+"/", "", -1)

	// Name without extension
	data.Name = strings.Replace(info.Name(), data.Extension, "", -1)
	data.NameSlug = slug.Make(data.Name)

	// Parse metadata
	metadataBytes := readMetadata(params, data)
	data.Metadata = parseExifMetadata(metadataBytes)
	data.MetadataDumpRaw = string(metadataBytes)
	var metadataOriginalArr []RawJsonMap;
	jsonerr := json.Unmarshal(metadataBytes, &metadataOriginalArr)

	if jsonerr == nil && len(metadataOriginalArr) > 0 {
		data.MetadataDump = metadataOriginalArr[0]
	}

	// Find file times
	data.ModificationTime = info.ModTime().Format(time.RFC3339)
	data.CreationTime = data.ModificationTime
	data.CreationTime = findEarliestCreationDate(data, params)

	// Find creation tool, camera, topic
	data.Topic = findFileTopic(path)
	data.CameraModel = findCameraName(data)
	data.CreationTool = findCreationTool(data)

	// Build Dest file name and dirName
	destDir, destName := buildDestPaths(data)
	data.dest.Name = destName
	data.dest.DirName = destDir
	data.dest.Extension = data.Extension
	data.dest.Path = params.Dest + "/" + data.dest.DirName + "/" + data.dest.Name + data.dest.Extension

	if pathExists(checksumPath(checksum, data.Extension, params.Dest)) || pathExists(data.dest.Path) {
		// Detect duplication by checksum or Dest path (e.g. when trying to copy twice from same folder)
		if filepath.Base(path) == filepath.Base(data.dest.Path) {
			// skip storing duplicate if same filename
			data.flags.skipped = true
		}
		data.flags.unique = false
		data.flags.duplicated = true
		return data, nil
	}

	return data, nil
}

func checksumPath(checksum, fileExtension, rootPath string) string {
	checksumRelPath := fmt.Sprintf("%s/%s/%s%s", checksum[0:2], checksum[2:3], checksum, fileExtension)
	return fmt.Sprintf("%s/%s/%s", rootPath, DirMetadata, checksumRelPath)
}

func buildDestPaths(data FileData) (string, string) {
	t, err := time.Parse(time.RFC3339, data.CreationTime)
	if err != nil {
		panic(err)
	}

	dateFolder := "2000/01/01"
	fileNamePrefix := "20000101-000000"
	topic := data.Topic

	if data.CameraModel != "" {
		topic = "cameras/" + slug.Make(data.CameraModel)
	}

	if data.MediaType != MediaTypeImages && topic == DefaultCameraModelFallback {
		dateFolder = fmt.Sprintf("%d/%02d", t.Year(), t.Month())
	} else {
		dateFolder = fmt.Sprintf("%d/%s/%02d", t.Year(), topic, t.Month())
	}

	fileNamePrefix = fmt.Sprintf("%d%02d%02d-%02d%02d%02d", t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())

	if !data.isMultimedia {
		fileNamePrefix += "-" + data.NameSlug
	}

	if strings.Contains(strings.ToUpper(fileNamePrefix), strings.ToUpper(data.Checksum)) {
		// Avoid repeating MD5 hash in the filename
		return data.MediaType + "/" + dateFolder, fileNamePrefix
	}

	return data.MediaType + "/" + dateFolder, fileNamePrefix + "-" + data.Checksum
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

	if data.Metadata.CreatorTool != "" {
		tool += data.Metadata.CreatorTool
	}

	if tool == "" && (data.Metadata.Software != "") {
		tool += data.Metadata.Software
	}

	return strings.TrimSpace(tool)
}

func findCameraName(data FileData) string {
	camera := ""

	if data.Metadata.Make != "" {
		camera = data.Metadata.Make
	}

	if data.Metadata.Model != "" {
		camera += " " + data.Metadata.Model
	}

	return strings.TrimSpace(camera)
}

func findEarliestCreationDate(data FileData, params appParams) string {
	var dates [6][2]string
	var foundDates []time.Time
	var creationDate time.Time

	metadataDateFormat := "2006:01:02 15:04:05" // 200601021504.05

	dates[0] = [2]string{data.ModificationTime, time.RFC3339}
	dates[1] = [2]string{data.Metadata.CreateDate, metadataDateFormat}
	dates[2] = [2]string{data.Metadata.DateTimeOriginal, metadataDateFormat}
	dates[3] = [2]string{data.Metadata.DateTimeDigitized, metadataDateFormat}
	dates[4] = [2]string{data.Metadata.GPSDateTime, metadataDateFormat + "Z"}
	dates[5] = [2]string{data.Metadata.FileModifyDate, metadataDateFormat + "+07:00"}

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
		return data.ModificationTime
	}

	return creationDate.Format(time.RFC3339)
}
