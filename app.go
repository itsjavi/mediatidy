package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

var LimitExceededError = errors.New("LIMIT EXCEEDED")

type appStats struct {
	Unique     int
	Duplicated int
	Skipped    int
	WithGPS    int
	Size       int64
}

type appParams struct {
	Src        string
	Dest       string
	Command    string
	Date       string
	Total      appStats
	Limit      *int
	Extensions *string
	FixDates   *bool
	// private:
	isAppDir bool // true if Src was created by this app
	move     bool
	dryRun   *bool
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
	params := appParams{Date: formatDate(time.Now(), ""), Total: appStats{Unique: 0, Duplicated: 0, Skipped: 0}}
	params.isAppDir = false
	params.Limit = flag.Int("limit", 0, "Limit the amount of processed files. 0 = no Limit.")
	params.dryRun = flag.Bool("dry-run", false, "If true, it won't do any write operations.")
	params.FixDates = flag.Bool("fix-dates", false, "If true, creation an modification times will be fixed in the file attributes.")
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

	if pathExists(params.Src + "/" + DirApp) {
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
	fileAppend(params.Dest+"/"+DirApp+"/"+AppName+".lsjson", fmt.Sprintf("%s", log)+"\n")
}

func logFileTransfer(file FileData, params appParams, destFile string) {
	logSameLn("%s ---> %s", file.relativePath, strings.Replace(destFile, params.Dest+"/", "", -1))
}

func storeFile(file FileData, params appParams) error {
	if file.flags.skipped {
		return nil
	}

	destDir := params.Dest + "/" + file.Dest.DirName
	destFileMeta := checksumPath(file.Checksum, file.Dest.Extension, params.Dest)

	if file.flags.duplicated {
		destDir = params.Dest + "/" + DirDuplicates + "/" + file.Dest.DirName
		destFileMeta = checksumPath(file.Checksum, file.Dest.Extension, params.Dest+"/"+DirDuplicates)
	}

	destFile := destDir + "/" + file.Dest.Name + file.Dest.Extension
	destDirMeta := path.Dir(destFileMeta)

	if *params.dryRun {
		logFileTransfer(file, params, destFile)
		return nil
	}

	if !pathExists(destDirMeta) {
		makeDir(destDirMeta)
	}

	makeDir(destDir)

	var err error

	if params.move {
		err = fileMove(file.Path, destFile)
	} else {
		err = fileCopy(file.Path, destFile, true)
	}

	if isError(err) {
		// Fatal on file copy/move errors
		catch(err)
	}

	if *params.FixDates {
		t, err := parseDate(time.RFC3339, file.CreationTime, file.Timezone)
		mt, err2 := parseDate(time.RFC3339, file.ModificationTime, file.Timezone)

		if !isError(err) && !isError(err2) {
			fileFixDates(destFile, t, mt)
		}
	}

	// Write meta file in the last step, to be sure the file has been moved/copied successfully before
	if !pathExists(destFileMeta) {
		meta, err := json.Marshal(file)
		if isError(err) {
			return err
		}
		err = ioutil.WriteFile(destFileMeta, meta, FilePerms)
		if isError(err) {
			return err
		}
	}

	return nil
}
