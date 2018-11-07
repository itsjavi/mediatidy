package core

import (
	"errors"
	"runtime"
)

const (
	AppName = "happytimes"

	FileSizeMin     = 1000 // 1000 B / 1 KB
	FileSizeMinDocs = 10   // 10 B (10 chars)

	DirApp        = "." + AppName
	DirMetadata   = DirApp + "/metadata"
	DirDuplicates = "_" + AppName + "_duplicates"

	CommandMove = "move"
	CommandCopy = "copy"
	CommandTest = "test"

	ExcludedDirRegex = "(?i)(\\.([a-z_0-9-]+)|/bower_components|/node_modules|/vendor|/Developer|/" + DirDuplicates + ")/.*$"
)

const (
	IsUnix = runtime.GOOS == "linux" || runtime.GOOS == "darwin" ||
		runtime.GOOS == "android" || runtime.GOOS == "freebsd" || runtime.GOOS == "openbsd"
)

type KeyValue map[string]string

type MediaType struct {
	Name        string
	Pattern     string
	MinFileSize int
	MaxFileSize int
}

type Category struct {
	Name    string
	Pattern string
}

type Config struct {
	MediaTypes      []MediaType
	Categories      []Category
	DefaultCategory string
}


type ContextStats struct {
	Unique     int
	Duplicated int
	Skipped    int
	WithGPS    int
	Size       int64
}

type Context struct {
	Src        string
	Dest       string
	Date       string
	Total      ContextStats
	Command    *string
	Limit      *int
	Extensions *string
	FixDates   *bool
	// private:
	isAppDir bool // true if Src was created by this app
}