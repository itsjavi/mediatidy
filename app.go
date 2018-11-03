package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

var LimitExceededError = errors.New("LIMIT EXCEEDED")
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
	src           string
	dest          string
	move          *bool
	limit         *int
	dryRun        *bool
	date          string
	total         appStats
	isUntangleDir bool // true if src was created by this app
	extensions    *string
}

func writeLogOutput(params appParams) {
	logVerb := "copied"

	if *params.move {
		logVerb = "moved"
	}

	logLn("%d files (+%d duplicates, %d skipped) %s from `%s` to `%s`",
		params.total.unique, params.total.duplicated, params.total.skipped, logVerb, params.src, params.dest)
}

func createAppParams() (appParams, error) {
	params := appParams{date: time.Now().Format(time.RFC3339), total: appStats{unique: 0, duplicated: 0, skipped: 0}}
	params.isUntangleDir = false
	params.limit = flag.Int("limit", 0, "Limit the amount of processed files. 0 = no limit.")
	params.move = flag.Bool("move", false, "Moves the files instead of copying them")
	params.dryRun = flag.Bool("dry-run", false, "If true, it won't do any write operations.")
	params.extensions = flag.String("ext", "", "Whitelist of file extensions to work with")

	flag.Parse()

	params.src = strings.TrimRight(strings.TrimSpace(flag.Arg(0)), string(os.PathSeparator))
	params.dest = strings.TrimRight(strings.TrimSpace(flag.Arg(1)), string(os.PathSeparator))

	if params.src == "" {
		return params, errors.New("missing argument 1: <src>")
	}

	if pathExists(params.src+"/"+AppLogFile) && pathExists(params.src+"/"+DirMetadata) {
		params.isUntangleDir = true
	}

	if params.dest == "" {
		params.dest = params.src + "-" + AppName
	}

	return params, nil
}

func writeLogFile(params appParams) {
	log, err := json.Marshal(params)
	catch(err)
	fileAppend(params.dest+"/"+AppLogFile, fmt.Sprintf("%s", log)+"\n")
}

func logFileTransfer(file FileData, params appParams, destFile string) {
	logSameLn("%s ---> %s", file.relPath, strings.Replace(destFile, params.dest+"/", "", -1))
}

func storeFile(file FileData, params appParams) error {
	destDir := params.dest + "/" + file.dest.dirName
	destDirMeta := params.dest + "/" + DirMetadata + "/" + file.dest.dirName
	destFileChecksum := checksumPath(file.mediaType, file.checksum, params.dest)

	if file.flags.duplicated {
		destDir = params.dest + "/" + DirDuplicates + "/" + file.dest.dirName
		destDirMeta = params.dest + "/" + DirDuplicates + "/" + DirMetadata + "/" + file.dest.dirName
		destFileChecksum = checksumPath(file.mediaType, file.checksum, params.dest+"/"+DirDuplicates)
	}

	destFile := destDir + "/" + file.dest.name + file.dest.extension
	destFileMeta := destDirMeta + "/" + file.dest.name + file.dest.extension + ".json"
	destFileTakeoutMeta := destDirMeta + "/" + file.dest.name + file.dest.extension + ".takeout.json"

	if *params.dryRun {
		logFileTransfer(file, params, destFile)
		return nil
	}

	if !pathExists(destFileChecksum) {
		relDestPath := strings.Replace(file.dest.path, params.dest + "/", "", -1)
		makeDir(path.Dir(destFileChecksum))
		err := ioutil.WriteFile(destFileChecksum, []byte(relDestPath), PathPerms)
		if err != nil {
			return err
		}
	}

	makeDir(destDirMeta)

	if !pathExists(destFileMeta) {
		err := ioutil.WriteFile(destFileMeta, []byte(file.metadataRaw), PathPerms)
		if err != nil {
			return err
		}
	}

	if !pathExists(destFileTakeoutMeta) && pathExists(file.path + ".json") {
		// Import Google Takeout metadata file
		err := fileCopy(file.path+".json",
			destDirMeta+"/"+file.dest.name+file.dest.extension+".takeout.json")
		if err != nil {
			return err
		}
	}

	makeDir(destDir)

	var err error

	if *params.move {
		err = fileMove(file.path, destFile)
	} else {
		err = fileCopy(file.path, destFile)
	}

	if err == nil {
		logFileTransfer(file, params, destFile)
	} else {
		catch(err)
	}

	return nil
}
