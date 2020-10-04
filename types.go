package main

import (
	"time"
)

type RawJsonMap map[string]interface{}

type CmdOptions struct {
	CurrentTime   time.Time // TODO: calculate elapsed time
	SrcDir        string
	DestDir       string
	DryRun        bool
	Limit         uint
	Extensions    string
	ConvertVideos bool
	FixDates      bool
	Move          bool
	Quiet         bool
}

type CmdFileStats struct {
	ProcessedFiles  int
	SkippedFiles    int
	DuplicatedFiles int
	TotalSize       int64
}

type FilePathInfo struct {
	Path      string
	Basename  string
	Dirname   string
	Extension string
}

