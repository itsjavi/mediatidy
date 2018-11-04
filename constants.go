package main

import "runtime"

const (
	AppName = "happybox"

	CommandMove = "move"
	CommandCopy = "copy"

	FileSizeMin     = 1000 // 1000 B / 1 KB
	FileSizeMinDocs = 10   // 10 B (10 chars)

	MediaTypeDocuments = "documents"
	MediaTypeAudios    = "audios"
	MediaTypeVideos    = "videos"
	MediaTypeImages    = "images"

	DefaultCameraModelFallback = "other"

	DirApp        = "." + AppName
	DirMetadata   = DirApp + "/metadata"
	DirDuplicates = "_" + AppName + "_duplicates"

	RegexImage       = "(?i)\\.(jpg|jpeg|gif|png|webp|tiff|bmp|raw|svg)$"
	RegexVideo       = "(?i)\\.(mpg|wmv|avi|mov|m4v|3gp|mp4|flv|webm|ogv|ts)$"
	RegexAudio       = "(?i)\\.(mp3|m4a|aac|wav|ogg|oga|wma|flac|opus|amr)$"
	RegexDocument    = "(?i)\\.(doc[x]?|xls[x]?|ppt[x]?|key|pages|numbers|md|pdf|zip|gz|7z|bak|psd|ai|afphoto|ics|mbox|vcf)$"
	RegexExcludeDirs = "(?i)(\\.([a-z_0-9-]+)|/bower_components|/node_modules|/vendor|/" + DirDuplicates + ")/.*$"

	DirPerms  = 0755
	FilePerms = 0644
	IsUnix    = runtime.GOOS == "linux" || runtime.GOOS == "darwin" ||
		runtime.GOOS == "android" || runtime.GOOS == "freebsd" || runtime.GOOS == "openbsd"
)
