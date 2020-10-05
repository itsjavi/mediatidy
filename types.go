package main

import (
	"path/filepath"
	"time"
)

type RawJsonMap map[string]interface{}

type AppContext struct {
	CurrentTime      time.Time // TODO: calculate elapsed time
	SrcDir           string
	DestDir          string
	DryRun           bool
	Limit            uint
	CustomExtensions string
	CustomMediaType  string
	CustomExclude    string
	MoveFiles        bool
	FixCreationDates bool
	Quiet            bool
	Db               DbHelper
	SrcDb            DbHelper
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

type FileImportStats struct {
	ProcessedFiles      int
	SkippedSameName     int
	SkippedSameChecksum int
	SkippedOther        int
	TotalSize           int64
}

type FilePathInfo struct {
	Path      string
	Basename  string
	Dirname   string
	Extension string
}
