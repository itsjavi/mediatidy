package main

import (
	"errors"
	tm "github.com/buger/goterm"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var StopWalk = errors.New("stop walking")

type TidyUpWalkFunc func(stats *WalkDirStats, path string, info os.FileInfo, err error) error

type AppContext struct {
	StartTime        time.Time // TODO: calculate elapsed time
	SrcDir           string
	DestDir          string
	DryRun           bool
	Limit            int
	CustomExtensions string
	CustomMediaType  string
	CustomExclude    string
	MoveFiles        bool
	FixCreationDates bool
	Quiet            bool
	Db               DbHelper
	SrcDb            DbHelper
}

type WalkDirStats struct {
	ProcessedFiles      int
	SkippedSameName     int
	SkippedSameChecksum int
	SkippedOther        int
	TotalSize           int64
}

func (ctx *AppContext) HasMetadataDb() bool {
	return PathExists(filepath.Join(ctx.DestDir, DirMetadata, DbFile))
}

func (ctx *AppContext) InitDb() {
	metadataDir := filepath.Join(ctx.DestDir, DirMetadata)
	MakeDirIfNotExists(metadataDir)

	ctx.Db.Init(filepath.Join(metadataDir, DbFile), true)
}

func (ctx *AppContext) HasSrcMetadataDb() bool {
	return PathExists(filepath.Join(ctx.SrcDir, DirMetadata, DbFile))
}

func (ctx *AppContext) InitSrcDbIfExists() bool {
	if !ctx.HasSrcMetadataDb() {
		return false
	}
	metadataDir := filepath.Join(ctx.SrcDir, DirMetadata)
	MakeDirIfNotExists(metadataDir)

	ctx.SrcDb.Init(filepath.Join(metadataDir, DbFile), true)
	return true
}

func tidyUpFile(ctx AppContext, stats *WalkDirStats, path string, info os.FileInfo, err error) (FileMeta, error) {
	Catch(err)

	fileData, err := GetFileMetadata(ctx, path, info)
	Catch(err)

	if fileData.IsDuplicationByDestBasename {
		stats.SkippedSameName++
		PrintReplaceLn("Skipped duplicate (file name): %s", path)
		return fileData, nil
	}

	if fileData.IsDuplicationByChecksum {
		stats.SkippedSameChecksum++
		PrintReplaceLn("Skipped duplicate (checksum): %s", path)
		return fileData, nil
	}

	stats.ProcessedFiles++
	stats.TotalSize += fileData.Size

	return fileData, processFile(ctx, fileData)
}

func TidyUp(ctx *AppContext) (WalkDirStats, error) {
	ctx.InitDb()
	ctx.InitSrcDbIfExists()

	return walkDir(*ctx, func(stats *WalkDirStats, path string, info os.FileInfo, err error) error {
		Catch(err)

		if ctx.Limit > 0 && stats.ProcessedFiles >= int(ctx.Limit) {
			return StopWalk
		}

		fileMeta, err := tidyUpFile(*ctx, stats, path, info, err)
		if IsError(err) {
			return err
		}

		if ctx.Quiet == false {
			printProgress(fileMeta.Origin.Path, *stats)
		}

		return nil
	})
}

func walkDir(ctx AppContext, processFileFunc TidyUpWalkFunc) (WalkDirStats, error) {
	stats := WalkDirStats{}
	return stats, filepath.Walk(ctx.SrcDir, func(path string, info os.FileInfo, err error) error {
		if IsError(err) {
			return err
		}

		if ctx.CustomExclude != "" {
			if regexp.MustCompile("(?i)(" + ctx.CustomExclude + ")/").MatchString(path) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		} else if regexp.MustCompile(RegexExcludeDirs).MatchString(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// File extension is in allowed list?
		if ctx.CustomExtensions != "" {
			if !regexp.MustCompile("(?i)\\.(" + ctx.CustomExtensions + ")$").MatchString(path) {
				PrintReplaceLn("Skipped not listed file extension: %s", path)
				stats.SkippedOther++
				return nil
			}
		} else if !regexp.MustCompile(RegexImage).MatchString(path) &&
			!regexp.MustCompile(RegexVideo).MatchString(path) {
			PrintReplaceLn("Skipped non media file: %s", path)
			stats.SkippedOther++
			return nil
		}

		fsize := info.Size()

		// File is too small?
		if fsize < int64(MinFileSize) {
			PrintReplaceLn("Skipped too small file: %s", path)
			stats.SkippedOther++
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

	if ctx.MoveFiles {
		Catch(FileMove(file.Origin.Path, destFile))
	} else {
		Catch(FileCopy(file.Origin.Path, destFile, true))
	}

	if ctx.FixCreationDates {
		ct, err := ParseDateWithTimezone(time.RFC3339, file.CreationDate, file.GPSTimezone)
		mt, err2 := ParseDateWithTimezone(time.RFC3339, file.ModificationDate, file.GPSTimezone)

		if !IsError(err) && !IsError(err2) {
			Catch(FileFixDates(destFile, ct, mt))
		}
	}

	// Write meta file in the last step, to be sure the file has been moved/copied successfully before

	file.OriginPath = getOriginPath(ctx, file)
	file.Path = filepath.Join(file.Destination.Dirname, file.Destination.Basename) + file.Destination.Extension
	file.Extension = file.Destination.Extension

	ctx.Db.InsertFileMetaIfNotExists(&file)

	return nil
}

// get origin path from SRC db instead (if exists)
func getOriginPath(ctx AppContext, file FileMeta) string {
	if ctx.HasSrcMetadataDb() {
		srcMeta, found, err := ctx.SrcDb.FindFileMetaByChecksum(file.Checksum)
		Catch(err)
		if found {
			return srcMeta.OriginPath
		}
	}
	return file.Origin.Path
}

func printProgress(currentFile string, stats WalkDirStats) {
	PrintReplaceLn(
		"[%s] "+tm.Color(tm.Bold("Stats: %s duplicates / %s skipped / %s processed / %s total size"), tm.YELLOW)+" / file: %s",
		AppName,
		ToString(stats.SkippedSameName+stats.SkippedSameChecksum),
		ToString(stats.SkippedOther),
		ToString(stats.ProcessedFiles),
		TotalBytesToString(stats.TotalSize, false),
		currentFile,
	)
}
