package app

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

type FileMeta struct {
	Source            FilePathInfo
	Destination       FilePathInfo
	MetadataPath      FilePathInfo
	Size              int64
	Checksum          string
	CreationTime      string
	ModificationTime  string
	MediaType         string
	CameraModel       string
	CreationTool      string
	IsScreenShot      bool
	IsDuplication     bool
	IsAlreadyImported bool
	IsLegacyVideo     bool
	Exif              ExifData
	GPS               GPSData
}
