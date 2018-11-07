package metadata

import (
	"encoding/json"
	"fmt"
	"github.com/bradfitz/latlong"
	"github.com/gosimple/slug"
	"github.com/itsjavi/happytimes/gps"
	"io/ioutil"
	"os"
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
type FileData struct {
	Path             string
	Dir              string
	RelativeDir      string
	Name             string
	NameSlug         string
	Dest             DestFile
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
	GPSPosition      gps.Position
	GPSTimezone      string
	Timezone         string
	// private:
	relativePath string
	isMultimedia bool
	flags        FileFlags
}

func getMediaType(ext string) string {
	if regexp.MustCompile(RegexContact).MatchString(ext) {
		return MediaTypeContacts
	}

	if regexp.MustCompile(RegexImage).MatchString(ext) {
		return MediaTypeImages
	}

	if regexp.MustCompile(RegexVideo).MatchString(ext) {
		return MediaTypeVideos
	}

	if regexp.MustCompile(RegexAudio).MatchString(ext) {
		return MediaTypeAudios
	}

	if regexp.MustCompile(RegexDocument).MatchString(ext) {
		return MediaTypeDocuments
	}

	if regexp.MustCompile(RegexArchive).MatchString(ext) {
		return MediaTypeArchives
	}

	return ""
}

func readMetadata(params appParams, file FileData) []byte {
	// Search for an already existing generated JSON metadata file (v2)
	pathsLookup := []string{
		// src, MD5
		checksumPath(file.Checksum, file.Extension, params.Src),
		// dest, MD5
		checksumPath(file.Checksum, file.Extension, params.Dest),
	}

	for _, srcMetaFile := range pathsLookup {
		if pathExists(srcMetaFile) {
			metadataBytes, err := ioutil.ReadFile(srcMetaFile)
			if !isError(err) {
				var meta FileData;
				jsonerr := json.Unmarshal(metadataBytes, &meta)
				if isError(jsonerr) {
					panic(jsonerr)
				}
				return []byte(meta.MetadataDumpRaw)
			}
		}
	}
	// Search for an already existing generated JSON metadata file (v1 metadata file)
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
			if !isError(err) {
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

	if data.MediaType != MediaTypeDocuments && (data.MediaType != MediaTypeArchives) {
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
	if isError(err) {
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

	// Parse metadata
	metadataBytes := readMetadata(params, data)
	data.Metadata = parseExifMetadata(metadataBytes)
	data.MetadataDumpRaw = string(metadataBytes)
	var metadataOriginalArr []RawJsonMap;
	jsonerr := json.Unmarshal(metadataBytes, &metadataOriginalArr)

	if !isError(jsonerr) && len(metadataOriginalArr) > 0 {
		data.MetadataDump = metadataOriginalArr[0]
	}

	data.Timezone = DefaultTimezone

	if data.Metadata.GPSPosition != "" {
		data.GPSPosition = parseGPSPosition(data.Metadata.GPSPosition)
		data.GPSTimezone = latlong.LookupZoneName(data.GPSPosition.Latitude, data.GPSPosition.Longitude)
		data.Timezone = data.GPSTimezone
	}

	var originalFilename string

	if data.Metadata.FileName == "" {
		originalFilename = data.Name
	} else {
		originalDir := strings.Replace(strings.Replace(
			filepath.Base(data.Metadata.Directory), "_", "", -1), "-", "", -1)

		if len(originalDir) >= 4 {
			originalFilename += strings.TrimSpace(originalDir) + "__"
		}

		originalFilename += strings.Replace(data.Metadata.FileName, "."+data.Metadata.FileTypeExtension, "", -1)
	}

	data.NameSlug = slug.Make(originalFilename)
	data.NameSlug = strings.Replace(data.NameSlug, "-", "_", -1)

	// Find file times
	data.ModificationTime = formatDate(info.ModTime(), "")
	data.CreationTime = data.ModificationTime
	data.CreationTime = findEarliestCreationDate(data, params)

	// Find creation tool, camera, topic
	data.CameraModel = findCameraName(data)
	data.CreationTool = findCreationTool(data)
	data.Topic = findTopic(data.Metadata.SourceFile + "/" + data.Metadata.CreatorTool + "/" + data.CameraModel)

	// Build Dest file name and dirName
	destDir, destName := buildDestPaths(data)
	data.Dest.Name = destName
	data.Dest.DirName = destDir
	data.Dest.Extension = data.Extension
	data.Dest.Path = params.Dest + "/" + data.Dest.DirName + "/" + data.Dest.Name + data.Dest.Extension

	isDuplicated := pathExists(checksumPath(checksum, data.Extension, params.Dest)) || pathExists(data.Dest.Path)

	if isDuplicated {
		// Detect duplication by checksum or Dest path (e.g. when trying to copy twice from same folder)
		if filepath.Base(path) == filepath.Base(data.Dest.Path) {
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
	return fmt.Sprintf("%s/%s/%s", rootPath, DirMetadata, checksumRelPath) + ".json"
}

func buildDestPaths(data FileData) (string, string) {
	t, err := time.Parse(time.RFC3339, data.CreationTime)
	if isError(err) {
		panic(err)
	}

	var dateFolder, destFilename, topic string

	topic = data.Topic

	if data.CameraModel != "" {
		topic = "cameras/" + slug.Make(data.CameraModel)
	}

	dateFolder = fmt.Sprintf("%d/%s/%02d", t.Year(), topic, t.Month())

	if data.MediaType != MediaTypeImages && topic == DefaultCameraModelFallback {
		dateFolder = fmt.Sprintf("%d/%02d", t.Year(), t.Month())
	}

	destFilename = fmt.Sprintf("%d%02d%02d-%02d%02d%02d", t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())

	if !strings.Contains(strings.ToUpper(data.NameSlug), strings.ToUpper(data.Checksum)) {
		// Avoid repeating MD5 hash in the filename
		destFilename += "-" + data.Checksum
	}

	if data.NameSlug != "" {
		destFilename += "-" + data.NameSlug
	}

	// Prevent path too long errors
	if len(destFilename) > 225 {
		destFilename = destFilename[0:224]
	}

	return data.MediaType + "/" + dateFolder, destFilename
}

func findTopic(haystack string) string {
	topic := DefaultCameraModelFallback
	tools := map[string]string{
		"music": `(?i)(music|itunes|songs|lyric|singing|karaoke|track|bgm|sound)`,
		//
		"screenshots": `(?i)(Screen Shot|Screenshot|Captura)`,
		//
		"facebook":  `(?i)(facebook)`,
		"instagram": `(?i)(instagram)`,
		"twitter":   `(?i)(twitter)`,
		"whatsapp":  `(?i)(whatsapp)`,
		"telegram":  `(?i)(telegram)`,
		"messenger": `(?i)(messenger)`,
		"snapchat":  `(?i)(snapchat)`,
	}

	for toolName, toolRegex := range tools {
		if regexp.MustCompile(toolRegex).MatchString(haystack) {
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

	if data.Metadata.Software != "" {
		tool += " " + data.Metadata.Software
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
	var dates [7][2]string
	var foundDates []time.Time
	var creationDate time.Time

	metadataDateFormat := "2006:01:02 15:04:05"

	dates[0] = [2]string{GetPhotoTakenTime(data, params.Src), DateFormat}
	dates[1] = [2]string{data.ModificationTime, DateFormat}
	dates[2] = [2]string{data.Metadata.CreateDate, metadataDateFormat}
	dates[3] = [2]string{data.Metadata.DateTimeOriginal, metadataDateFormat}
	dates[4] = [2]string{data.Metadata.DateTimeDigitized, metadataDateFormat}
	dates[5] = [2]string{data.Metadata.GPSDateTime, metadataDateFormat + "Z"}
	dates[6] = [2]string{data.Metadata.FileModifyDate, metadataDateFormat + "+07:00"}

	for _, val := range dates {
		if val[0] == "" {
			continue
		}
		t, err := time.Parse(val[1], val[0])
		if isError(err) {
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

	return formatDate(creationDate, data.Timezone)
}
