package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gosimple/slug"
)

/*
PHASES:
1. Parse CLI arguments
2. Walk path
3. Check if path is file and readable
4. Check if file is supported type
5. Check if file is not too small
6. Extract Metadata
7. Build destination path name (media type, camera, date)
8. Build destination filename (date/slug, file md5)
9. Detect if absolute file destination path is a duplicate
10. Move/copy file
11. Create metadata JSON file
12. Add entry into log file
 */

var imageReg = regexp.MustCompile(RegexImage)
var videoReg = regexp.MustCompile(RegexVideo)
var audioReg = regexp.MustCompile(RegexAudio)
var docsReg = regexp.MustCompile(RegexDocument)

type appStats struct {
	unique     int
	duplicated int
	skipped    int
}

type appParams struct {
	src      string
	dest     string
	move     *bool
	limit    *int
	date     string
	total    appStats
	srcByApp bool // true if src was created by this app
}

type fileFlags struct {
	unique     bool
	duplicated bool
	skipped    bool
}

type destFile struct {
	path      string
	name      string
	dirName   string
	extension string
}

type fileData struct {
	path             string
	dest             destFile
	name             string
	nameSlug         string
	size             int64
	extension        string
	creationTime     string
	modificationTime string
	mediaType        string
	isMultimedia     bool
	flags            fileFlags
	metadata         ExifToolData
	metadataRaw      string
	creationTool     string
	cameraModel      string
	topic            string
	checksum         string
}

func createAppParams() (appParams, error) {
	params := appParams{date: time.Now().Format(time.RFC3339), total: appStats{unique: 0, duplicated: 0, skipped: 0}}
	params.srcByApp = false
	params.limit = flag.Int("limit", 0, "Limit the amount of processed files")
	params.move = flag.Bool("move", false, "Moves the files instead of copying them")

	flag.Parse()

	params.src = strings.TrimRight(strings.TrimSpace(flag.Arg(0)), string(os.PathSeparator))
	params.dest = strings.TrimRight(strings.TrimSpace(flag.Arg(1)), string(os.PathSeparator))

	if params.src == "" {
		return params, errors.New("missing argument 1: <src>")
	}

	if pathExists(params.src+"/"+AppLogFile) && pathExists(params.src+"/"+DirMetadata) {
		params.srcByApp = true
	}

	if params.dest == "" {
		params.dest = params.src + "-" + AppName
	}

	return params, nil
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

func parseMetadata(params appParams, file fileData) []byte {
	if params.srcByApp {
		fileRelDir := filepath.Dir(strings.Replace(file.path, params.src+"/", "", -1))
		srcMetaFile := params.src + "/" + DirMetadata + "/" + fileRelDir + "/" + file.name + ".json"

		if pathExists(srcMetaFile) {
			metadataBytes, err := ioutil.ReadFile(srcMetaFile)
			if err == nil {
				return metadataBytes
			}
		}
	}

	fallbackMetadata := []byte(`[{"SourceFile":"` + file.path + `", "Error": true}]`)
	return extractMetadata(file.path, fallbackMetadata)
}

func parseFileData(params appParams, path string, info os.FileInfo) (fileData, error) {
	data := fileData{path: path, size: info.Size(), flags: fileFlags{skipped: false, duplicated: false, unique: false}}

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

	// Name without extension
	data.name = strings.Replace(info.Name(), data.extension, "", -1)
	data.nameSlug = slug.Make(data.name)

	// Find file times
	data.modificationTime = info.ModTime().Format(time.RFC3339)
	data.creationTime = findEarliestCreationDate(data)

	// Parse metadata
	metadataBytes := parseMetadata(params, data)
	data.metadata = unmarshalMetadata(metadataBytes)
	data.metadataRaw = string(metadataBytes)

	data.flags.unique = true

	// MD5 checksum
	checksum, err := getFileChecksum(data.path)
	if err != nil {
		data.flags.skipped = true
		return data, err
	}
	data.checksum = checksum

	// Find creation tool, camera, topic
	data.topic = findFileTopic(path)
	data.cameraModel = findCameraName(data)
	data.creationTool = findCreationTool(data)

	// Build dest file name and dirName
	destDir, destName := buildDestPaths(data)
	data.dest.name = destName
	data.dest.dirName = destDir
	data.dest.extension = data.extension
	data.dest.path = params.dest + "/" + data.dest.dirName + "/" + data.dest.name + data.dest.extension

	// Detect duplication
	if pathExists(data.dest.path) {
		data.flags.unique = false
		data.flags.duplicated = true
	}

	return data, nil
}

func buildDestPaths(data fileData) (string, string) {
	t, err := time.Parse(time.RFC3339, data.creationTime)
	dateFolder := "2000/01/01"
	fileNamePrefix := "20000101-000000"
	topic := data.topic

	if data.cameraModel != "" {
		topic = data.cameraModel
	}

	if err == nil {
		if data.mediaType != MediaTypeImages && topic == DefaultCameraModelFallback {
			dateFolder = fmt.Sprintf("%d/%02d", t.Year(), t.Month())
		} else {
			dateFolder = fmt.Sprintf("%d/%s/%02d", t.Year(), slug.Make(topic), t.Month())
		}
	}

	if data.isMultimedia && (err == nil) {
		fileNamePrefix = fmt.Sprintf("%d%02d%02d-%02d%02d%02d", t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second())
	} else if data.isMultimedia {
		fileNamePrefix = data.nameSlug
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

func findCreationTool(data fileData) string {
	tool := ""

	if data.metadata.CreatorTool != "" {
		tool += data.metadata.CreatorTool
	}

	if tool == "" && (data.metadata.Software != "") {
		tool += data.metadata.Software
	}

	return strings.TrimSpace(tool)
}

func findCameraName(data fileData) string {
	camera := ""

	if data.metadata.Make != "" {
		camera = data.metadata.Make
	}

	if data.metadata.Model != "" {
		camera += " " + data.metadata.Model
	}

	return strings.TrimSpace(camera)
}

func findEarliestCreationDate(data fileData) string {
	var dates [6][2]string
	var foundDates []time.Time
	var creationDate time.Time

	data.creationTime = data.modificationTime

	metadataDateFormat := "2006:01:02 15:04:05"

	dates[0] = [2]string{data.creationTime, time.RFC3339}
	dates[1] = [2]string{data.metadata.CreateDate, metadataDateFormat}
	dates[2] = [2]string{data.metadata.DateTimeOriginal, metadataDateFormat}
	dates[3] = [2]string{data.metadata.DateTimeDigitized, metadataDateFormat}
	dates[4] = [2]string{data.modificationTime, time.RFC3339}
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

		if (val.Year() > 1970) && (val.Unix() < creationDate.Unix()) {
			creationDate = val
		}
	}

	if creationDate.IsZero() {
		return data.modificationTime
	}

	return creationDate.Format(time.RFC3339)
}

func writeLogFile(params appParams) {
	log, err := json.Marshal(params)
	catch(err)
	fileAppend(params.dest+"/"+AppLogFile, fmt.Sprintf("%s", log)+"\n")
}

func storeFile(file fileData, params appParams) error {
	// logln("saving %s", file.path)
	destDir := params.dest + "/" + file.dest.dirName
	destDirMeta := params.dest + "/" + DirMetadata + "/" + file.dest.dirName

	if file.flags.duplicated {
		destDir = params.dest + "/" + DirDuplicates + "/" + file.dest.dirName
		destDirMeta = params.dest + "/" + DirDuplicates + "/" + DirMetadata + "/" + file.dest.dirName
	}

	destFile := destDir + "/" + file.dest.name + file.dest.extension
	destFileMeta := destDirMeta + "/" + file.dest.name + ".json"

	if !pathExists(destFileMeta) {
		mkdirp(destDirMeta)
		err := ioutil.WriteFile(destFileMeta, []byte(file.metadataRaw), PathPerms)
		if err != nil {
			return err
		}
	}

	mkdirp(destDir)

	if *params.move {
		panic("Moving is disabled for now, until the code gets stable.")
		//return mv(file.path, destFile)
	} else {
		return cp(file.path, destFile)
	}
}

func writeLogOutput(params appParams) {
	logVerb := "copied"

	if *params.move {
		logVerb = "moved"
	}

	logln("%d files (+%d duplicates, %d skipped) %s from `%s` to `%s`",
		params.total.unique, params.total.duplicated, params.total.skipped, logVerb, params.src, params.dest)
}

func logsameln(format string, args ... interface{}) {
	fmt.Printf("\033[2K\r"+format, args...)
}

var LimitExceededError = errors.New("LIMIT EXCEEDED")

func main() {
	params, err := createAppParams()
	catch(err)

	fmt.Printf("\n")

	err = filepath.Walk(params.src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && !strings.ContainsAny(path, "#@") {
			scannedDir := strings.Replace(path, params.src+"/", "", -1)
			if len(scannedDir) > 100 {
				scannedDir = scannedDir[0:99] + "..."
			}
			logsameln(">> Analyzing: %s", scannedDir)
		}

		if info.IsDir() {
			return nil
		}

		if regexp.MustCompile(RegexExcludeDirs).MatchString(path) {
			params.total.skipped++
			return nil
		}

		processed := params.total.unique + params.total.duplicated

		if (*params.limit > 0) && (processed >= *params.limit) {
			return LimitExceededError
		}

		// logln("Visiting %s", path)
		data, err := parseFileData(params, path, info)

		//logsameln(">> Processing # %s : %s", strconv.Itoa(processed+1), relPath)

		if err != nil {
			return err
		}

		if data.flags.skipped {
			params.total.skipped++
			// logln("    Skipped %s", data.path)
			return nil
		}

		err = storeFile(data, params)
		if err != nil {
			return err
		}

		if data.flags.unique {
			params.total.unique++
		}

		if data.flags.duplicated {
			params.total.duplicated++
		}

		// jsonData, e := json.Marshal(data)
		// fmt.Printf("%+v \n", data)

		return nil
	})

	fmt.Printf("\n\n")

	if err != LimitExceededError {
		catch(err)
	}
	writeLogFile(params)
	writeLogOutput(params)
}
