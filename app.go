package main

import (
	tm "github.com/buger/goterm"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"
)

type TidyUpWalkFunc func(stats *CmdFileStats, path string, info os.FileInfo, err error) error

func tidyUpFile(params CmdOptions, stats *CmdFileStats, path string, info os.FileInfo, err error) (FileMeta, error) {
	HandleError(err)

	fileData, err := GetFileMetadata(params, path, info)
	HandleError(err)

	if fileData.IsAlreadyImported {
		stats.SkippedFiles++
		return fileData, nil
	}

	if fileData.IsDuplication {
		stats.SkippedFiles++
		stats.DuplicatedFiles++
		return fileData, nil
	}

	stats.ProcessedFiles++
	stats.TotalSize += fileData.Size

	return fileData, processFile(params, fileData)
}

func TidyUp(params CmdOptions) (CmdFileStats, error) {
	return walkDir(params, func(stats *CmdFileStats, path string, info os.FileInfo, err error) error {
		HandleError(err)

		fileMeta, err := tidyUpFile(params, stats, path, info, err)
		if IsError(err) {
			return err
		}

		if params.Quiet == false {
			printProgress(fileMeta, *stats)
		}

		return nil
	})
}

func walkDir(params CmdOptions, processFileFunc TidyUpWalkFunc) (CmdFileStats, error) {
	stats := CmdFileStats{}
	return stats, filepath.Walk(params.SrcDir, func(path string, info os.FileInfo, err error) error {
		if IsError(err) {
			return err
		}

		if regexp.MustCompile(RegexExcludeDirs).MatchString(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if !regexp.MustCompile(RegexImage).MatchString(path) &&
			!regexp.MustCompile(RegexVideo).MatchString(path) {
			stats.SkippedFiles++
			return nil
		}

		fsize := info.Size()

		// File is too small?
		if fsize < int64(MinFileSize) {
			stats.SkippedFiles++
			return nil
		}

		// File extension is in allowed list?
		if params.Extensions != "" && !regexp.MustCompile("(?i)\\.("+params.Extensions+")$").MatchString(path) {
			stats.SkippedFiles++
			return nil
		}

		return processFileFunc(&stats, path, info, err)
	})
}

func processFile(params CmdOptions, file FileMeta) error {
	destDir := params.DestDir + "/" + file.Destination.Dirname
	destFile := destDir + "/" + file.Destination.Basename + file.Destination.Extension

	destFileMeta := file.MetadataPath.Path
	destDirMeta := path.Dir(destFileMeta)

	if params.DryRun {
		return nil
	}

	MakeDirIfNotExists(destDirMeta)
	MakeDirIfNotExists(destDir)

	// TODO: convert videos

	if params.Move {
		HandleError(FileMove(file.Source.Path, destFile))
	} else {
		HandleError(FileCopy(file.Source.Path, destFile, true))
	}

	if params.FixDates {
		ct, err := ParseDateWithTimezone(time.RFC3339, file.CreationTime, file.GPS.Timezone)
		mt, err2 := ParseDateWithTimezone(time.RFC3339, file.ModificationTime, file.GPS.Timezone)

		if !IsError(err) && !IsError(err2) {
			HandleError(FileFixDates(destFile, ct, mt))
		}
	}

	// Write meta file in the last step, to be sure the file has been moved/copied successfully before
	if !PathExists(destFileMeta) {
		meta, err := JsonEncodePretty(file)
		if IsError(err) {
			return err
		}
		err = ioutil.WriteFile(destFileMeta, meta, FilePerms)
		if IsError(err) {
			return err
		}
	}

	return nil
}

func printProgress(currentFile FileMeta, stats CmdFileStats) {
	PrintReplaceLn(
		"[%s] "+tm.Color(tm.Bold("Stats: %s duplicates / %s skipped / %s processed / %s total size"), tm.YELLOW)+" / file: %s",
		AppName,
		ToString(stats.DuplicatedFiles),
		ToString(stats.SkippedFiles),
		ToString(stats.ProcessedFiles),
		TotalBytesToString(stats.TotalSize, false),
		currentFile.Source.Path,
	)
}
