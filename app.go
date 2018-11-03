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
	Unique     int
	Duplicated int
	Skipped    int
}

type appParams struct {
	Src        string
	Dest       string
	move       bool
	Command    string
	Date       string
	Total      appStats
	isAppDir   bool // true if Src was created by this app
	Limit      *int
	dryRun     *bool
	Extensions *string
}

func writeLogOutput(params appParams) {
	logVerb := "copied"

	if params.move {
		logVerb = "moved"
	}

	logLn("%d files (+%d duplicates, %d Skipped) %s from `%s` to `%s`",
		params.Total.Unique, params.Total.Duplicated, params.Total.Skipped, logVerb, params.Src, params.Dest)
}

func createAppParams() (appParams, error) {
	params := appParams{Date: time.Now().Format(time.RFC3339), Total: appStats{Unique: 0, Duplicated: 0, Skipped: 0}}
	params.isAppDir = false
	params.Limit = flag.Int("limit", 0, "Limit the amount of processed files. 0 = no Limit.")
	params.dryRun = flag.Bool("dry-run", false, "If true, it won't do any write operations.")
	params.Extensions = flag.String("ext", "", "Whitelist of file Extensions to work with")

	flag.Parse()

	params.Command = strings.TrimRight(strings.TrimSpace(flag.Arg(0)), string(os.PathSeparator))
	params.Src = strings.TrimRight(strings.TrimSpace(flag.Arg(1)), string(os.PathSeparator))
	params.Dest = strings.TrimRight(strings.TrimSpace(flag.Arg(2)), string(os.PathSeparator))

	switch params.Command {
	case CommandMove:
		params.move = true
		break
	case CommandCopy:
		params.move = false
	default:
		return params, errors.New("invalid Command. Supported commands are: move, copy")
	}

	if params.Src == "" {
		return params, errors.New("missing argument 2: <Src>")
	}

	if pathExists(params.Src + "/" + DirMetadata) {
		params.isAppDir = true
	}

	if params.Dest == "" {
		params.Dest = params.Src + "-" + AppName
	}

	return params, nil
}

func writeLogFile(params appParams) {
	log, err := json.Marshal(params)
	catch(err)
	fileAppend(params.Dest+"/"+DirMetadata+"/"+AppName+".lsjson", fmt.Sprintf("%s", log)+"\n")
}

func logFileTransfer(file FileData, params appParams, destFile string) {
	logSameLn("%s ---> %s", file.relPath, strings.Replace(destFile, params.Dest+"/", "", -1))
}

func storeFile(file FileData, params appParams) error {
	destDir := params.Dest + "/" + file.dest.dirName
	destDirMeta := params.Dest + "/" + DirMetadata + "/" + file.dest.dirName
	destFileChecksum := checksumPath(file.mediaType, file.checksum, params.Dest)

	if file.flags.duplicated {
		destDir = params.Dest + "/" + DirDuplicates + "/" + file.dest.dirName
		destDirMeta = params.Dest + "/" + DirDuplicates + "/" + DirMetadata + "/" + file.dest.dirName
		destFileChecksum = checksumPath(file.mediaType, file.checksum, params.Dest+"/"+DirDuplicates)
	}

	destFile := destDir + "/" + file.dest.name + file.dest.extension
	destFileMeta := destDirMeta + "/" + file.dest.name + file.dest.extension + ".json"
	destFileTakeoutMeta := destDirMeta + "/" + file.dest.name + file.dest.extension + ".takeout.json"

	if *params.dryRun {
		logFileTransfer(file, params, destFile)
		return nil
	}

	if !pathExists(destFileChecksum) {
		relDestPath := strings.Replace(file.dest.path, params.Dest+"/", "", -1)
		makeDir(path.Dir(destFileChecksum))
		err := ioutil.WriteFile(destFileChecksum, []byte(relDestPath), FilePerms)
		if err != nil {
			return err
		}
	}

	makeDir(destDirMeta)

	if !pathExists(destFileMeta) {
		err := ioutil.WriteFile(destFileMeta, []byte(file.metadataRaw), FilePerms)
		if err != nil {
			return err
		}
	}

	if !pathExists(destFileTakeoutMeta) && pathExists(file.path+".json") {
		// Import Google Takeout metadata file
		err := fileCopy(file.path+".json",
			destDirMeta+"/"+file.dest.name+file.dest.extension+".takeout.json", true)
		if err != nil {
			return err
		}
	}

	makeDir(destDir)

	var err error

	if params.move {
		err = fileMove(file.path, destFile)
	} else {
		err = fileCopy(file.path, destFile, true)
	}

	if err != nil {
		catch(err)
	}

	return nil
}
