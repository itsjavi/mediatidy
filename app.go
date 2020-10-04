package main

import (
	tm "github.com/buger/goterm"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type TidyUpWalkFunc func(stats *FileImportStats, path string, info os.FileInfo, err error) error

func tidyUpFile(ctx AppContext, stats *FileImportStats, path string, info os.FileInfo, err error) (FileMeta, error) {
	HandleError(err)

	fileData, err := GetFileMetadata(ctx, path, info)
	HandleError(err)

	if fileData.IsDuplicationByDestBasename {
		stats.SkippedFiles++
		return fileData, nil
	}

	if fileData.IsDuplicationByChecksum {
		stats.SkippedFiles++
		stats.DuplicatedFiles++
		return fileData, nil
	}

	stats.ProcessedFiles++
	stats.TotalSize += fileData.Size

	return fileData, processFile(ctx, fileData)
}

func TidyUp(ctx *AppContext) (FileImportStats, error) {
	ctx.InitDb()
	ctx.InitSrcDbIfExists()

	return walkDir(*ctx, func(stats *FileImportStats, path string, info os.FileInfo, err error) error {
		HandleError(err)

		fileMeta, err := tidyUpFile(*ctx, stats, path, info, err)
		if IsError(err) {
			return err
		}

		if ctx.Quiet == false {
			printProgress(fileMeta, *stats)
		}

		return nil
	})
}

func walkDir(ctx AppContext, processFileFunc TidyUpWalkFunc) (FileImportStats, error) {
	stats := FileImportStats{}
	return stats, filepath.Walk(ctx.SrcDir, func(path string, info os.FileInfo, err error) error {
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
		if ctx.Extensions != "" && !regexp.MustCompile("(?i)\\.("+ctx.Extensions+")$").MatchString(path) {
			stats.SkippedFiles++
			return nil
		}

		return processFileFunc(&stats, path, info, err)
	})
}

func processFile(ctx AppContext, file FileMeta) error {
	destDir := ctx.DestDir + "/" + file.Destination.Dirname
	destFile := destDir + "/" + file.Destination.Basename + file.Destination.Extension

	if ctx.DryRun {
		return nil
	}

	MakeDirIfNotExists(destDir)

	// TODO: convert videos

	if ctx.Move {
		HandleError(FileMove(file.Origin.Path, destFile))
	} else {
		HandleError(FileCopy(file.Origin.Path, destFile, true))
	}

	if ctx.FixDates {
		ct, err := ParseDateWithTimezone(time.RFC3339, file.CreationDate, file.GPSTimezone)
		mt, err2 := ParseDateWithTimezone(time.RFC3339, file.ModificationDate, file.GPSTimezone)

		if !IsError(err) && !IsError(err2) {
			HandleError(FileFixDates(destFile, ct, mt))
		}
	}

	// Write meta file in the last step, to be sure the file has been moved/copied successfully before

	file.OriginPath = getOriginPath(ctx, file)
	file.Path = filepath.Join(file.Destination.Dirname, file.Destination.Basename, file.Destination.Extension)
	file.Extension = file.Destination.Extension
	file.ExifJson = file.Exif.FullJsonDump

	ctx.Db.InsertFileMetaIfNotExists(&file)

	return nil
}

// get origin path from SRC db instead (if exists)
func getOriginPath(ctx AppContext, file FileMeta) string {
	if ctx.HasSrcMetadataDb() {
		srcMeta, found, err := ctx.SrcDb.FindFileMetaByChecksum(file.Checksum)
		HandleError(err)
		if found {
			return srcMeta.OriginPath
		}
	}
	return file.Origin.Path
}

func printProgress(currentFile FileMeta, stats FileImportStats) {
	PrintReplaceLn(
		"[%s] "+tm.Color(tm.Bold("Stats: %s duplicates / %s skipped / %s processed / %s total size"), tm.YELLOW)+" / file: %s",
		AppName,
		ToString(stats.DuplicatedFiles),
		ToString(stats.SkippedFiles),
		ToString(stats.ProcessedFiles),
		TotalBytesToString(stats.TotalSize, false),
		currentFile.Origin.Path,
	)
}
