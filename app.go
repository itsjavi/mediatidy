package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"
)

type TidyUpWalkFunc func(stats *CmdFileStats, path string, info os.FileInfo, err error) error

func TidyUp(params CmdOptions) (CmdFileStats, error) {
	return walkDir(params, func(stats *CmdFileStats, path string, info os.FileInfo, err error) error {
		fileData, err := GetFileMetadata(params, path, info)
		HandleError(err)

		if fileData.IsAlreadyImported {
			stats.SkippedFiles++
			return nil
		}

		if fileData.IsDuplication {
			stats.SkippedFiles++
			stats.DuplicatedFiles++
			return nil
		}

		stats.ProcessedFiles++
		stats.TotalSize += fileData.Size

		return processFile(params, fileData)
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

func logFileTransfer(file FileMeta, prefix string) {
	PrintReplaceLn(
		"%s%s ---> %s",
		prefix,
		file.Source.Dirname+"/"+file.Source.Basename+file.Source.Extension,
		file.Destination.Dirname+"/"+file.Destination.Basename+file.Destination.Extension,
	)
}

func processFile(params CmdOptions, file FileMeta) error {
	destDir := params.DestDir + "/" + file.Destination.Dirname
	destFile := destDir + "/" + file.Destination.Basename + file.Destination.Extension

	destFileMeta := file.MetadataPath.Path
	destDirMeta := path.Dir(destFileMeta)

	if params.DryRun {
		logFileTransfer(file, AppName+" [dry-drun] ")
		return nil
	}

	MakeDirIfNotExists(destDirMeta)
	MakeDirIfNotExists(destDir)

	// TODO: convert videos

	if params.Move {
		HandleError(FileMove(file.Source.Path, destFile))
		logFileTransfer(file, AppName+" [moving] ")
	} else {
		HandleError(FileCopy(file.Source.Path, destFile, true))
		logFileTransfer(file, AppName+" [copying] ")
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

func printStats(stats CmdFileStats) {
	fmt.Println("\nStats: ")
	fmt.Printf("	- Duplicated Files: %s\n", ToString(stats.DuplicatedFiles))
	fmt.Printf("	- Skipped Files: %s\n", ToString(stats.SkippedFiles))
	fmt.Printf("	- Processed Files: %s\n", ToString(stats.ProcessedFiles))
	fmt.Printf("	- Total Processed Size: %s\n", TotalBytesToString(stats.TotalSize, false))
}
