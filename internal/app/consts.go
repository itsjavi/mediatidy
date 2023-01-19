package app

import (
	"runtime"
	"time"
)

const (
	AppName = "mediatidy"

	IsUnix = runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" || runtime.GOOS == "openbsd"

	MinFileSize = 1000 // 1000 B / 1 KB
	DirPerms    = 0755
	FilePerms   = 0644

	DirMetadata        = ".metadata"
	DirVideos          = "originals"
	DirImages          = "originals"
	DirVideosConverted = "converted"

	MediaTypeVideo = "video"
	MediaTypeImage = "image"

	DateFormat          = time.RFC3339
	DateTimestampFormat = "2006:01:02 15:04:05"
	DefaultTimezone     = "Europe/Berlin"

	DefaultCameraModelFallback = "Unknown"

	RegexImage       = "(?i)\\.(jpg|jpeg|gif|png|heic|heif|webp|tiff|tif|bmp|raw|svg|psd|ai)$"
	RegexVideo       = "(?i)\\.(mpg|wmv|avi|mov|m4v|3gp|mp4|flv|webm|ogv|ts|divx|mkv|mpeg)$"
	RegexVideoOld    = "(?i)\\.(mpg|wmv|avi|mov|m4v|3gp|flv|divx|mpeg)$"
	RegexExcludeDirs = "(?i)(\\.([a-z_0-9-]+)|/bower_components|/node_modules|/vendor|/Developer)/.*$"
	RegexScreenShot  = "(?i)(Screen Shot|Screen Record|Screenshot|Captur)"
)
