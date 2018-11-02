package main

const (
	AppName    = "untangle"
	AppLogFile = ".untangle.json"

	FileSizeMin     = 1000 // 1 kb
	FileSizeMinDocs = 10   // 10 B (10 chars)

	MediaTypeDocuments = "documents"
	MediaTypeAudios    = "audios"
	MediaTypeVideos    = "videos"
	MediaTypeImages    = "images"

	DefaultCameraModelFallback = "no-camera/other"

	DirDuplicates = ".duplicates"
	DirMetadata   = ".metadata"

	// TODO: add possibility to specify extensions via command line options
	RegexImage       = "(?i)\\.(jpg|jpeg|gif|png|webp|tiff|bmp|raw|svg)$"
	RegexVideo       = "(?i)\\.(mpg|wmv|avi|mov|m4v|3gp|mp4|flv|webm|ogv|ts)$"
	RegexAudio       = "(?i)\\.(mp3|m4a|aac|wav|ogg|oga|wma|flac|opus|amr)$"
	RegexDocument    = "(?i)\\.(doc[x]?|xls[x]?|md|pdf|zip|gz|7z|bak|psd|ai|afphoto|ics|mbox|vcf)$"
	RegexExcludeDirs = "(?i)(\\.([a-z_0-9-]+)|/bower_components|/node_modules|/developer)/.*$"

	PathPerms = 0755
)
