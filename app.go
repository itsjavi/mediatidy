package main

import (
	"fmt"
	tm "github.com/buger/goterm"
	"github.com/disintegration/imaging"
	"github.com/kalafut/imohash"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const IsUnix = runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" || runtime.GOOS == "openbsd"
const (
	AppName                    = "mediatidy"
	MinFileSize                = 1000 // 1000 B / 1 KB
	DirPerms                   = 0755
	FilePerms                  = 0644
	MetadataDbFile             = "metadata.sqlite"
	DirDatabases               = "databases"
	DirOriginals               = "originals"
	DirThumbnails              = "thumbnails"
	ThumbnailWidth             = 480 // 1920/4
	ThumbnailHeight            = 270 // 1080/4
	PortraitThumbnailWidth     = ThumbnailHeight
	PortraitThumbnailHeight    = ThumbnailWidth
	GifMaxDuration             = 2
	GifFrameRate               = 10
	RegexImage                 = "(?i)\\.(jpg|jpeg|gif|png)$"
	RegexVideo                 = "(?i)\\.(mpg|wmv|avi|mov|m4v|3gp|mp4|flv|webm|ogv|ts|divx|mkv|mpeg)$"
	RegexExcludeDirs           = "(?i)(\\.([a-z_0-9-]+)|/bower_components|/node_modules|/vendor|/Developer|/[Tt]humbnail[s]?)/.*$"
	RegexScreenShot            = "(?i)(Screen Shot|Screen Record|Screenshot|Captur)"
	ThumnailableImageMimeTypes = "(?i)(^image/(jpeg|png|gif)$)"
	// RegexVideoOld    = "(?i)\\.(mpg|wmv|avi|mov|m4v|3gp|flv|divx|mpeg)$"
)

type AppContext struct {
	StartTime        time.Time // TODO: calculate elapsed time
	SrcDir           string
	DestDir          string
	DryRun           bool
	Limit            int
	CustomExtensions string
	CustomMediaType  string
	CustomExclude    string
	CreateDbOnly     bool
	CreateThumbnails bool
	MoveFiles        bool
	FixCreationDates bool
	Quiet            bool
	Db               DbHelper
	SrcDb            DbHelper
	Exiftool         *Exiftool
}

type AppRunStats struct {
	ProcessedFiles      int
	SkippedSameName     int
	SkippedSameChecksum int
	SkippedOther        int
	TotalSize           int64
}

func (ctx *AppContext) OpenExiftool() {
	exifTool := Exiftool{}
	exifTool.UseDefaults()
	Catch(exifTool.Open())
	ctx.Exiftool = &exifTool
}

func (ctx *AppContext) HasMetadataDb() bool {
	return PathExists(filepath.Join(ctx.DestDir, DirDatabases, MetadataDbFile))
}

func (ctx *AppContext) HasSrcMetadataDb() bool {
	return PathExists(filepath.Join(ctx.SrcDir, DirDatabases, MetadataDbFile))
}

func (ctx *AppContext) InitDb() {
	metadataDir := filepath.Join(ctx.DestDir, DirDatabases)
	MakeDirIfNotExists(metadataDir)

	ctx.Db.Init(filepath.Join(metadataDir, MetadataDbFile), true)
}

func (ctx *AppContext) InitSrcDbIfExists() bool {
	if !ctx.HasSrcMetadataDb() {
		return false
	}
	metadataDir := filepath.Join(ctx.SrcDir, DirDatabases)
	MakeDirIfNotExists(metadataDir)

	ctx.SrcDb.Init(filepath.Join(metadataDir, MetadataDbFile), false)

	return true
}

func (ctx *AppContext) CloseDb() {
	Catch(ctx.Db.Close())
}

func (ctx *AppContext) CloseSrcDbIfExists() {
	if !ctx.HasSrcMetadataDb() {
		return
	}
	Catch(ctx.SrcDb.Close())
}

func TidyRoutine(ctx AppContext, stats *AppRunStats, fileMetaChann chan FileMeta) {
	walkChan := WalkDirRoutine(ctx)
	ctx.OpenExiftool()
	ctx.InitSrcDbIfExists()
	ctx.InitDb()

	defer ctx.Exiftool.Close()
	defer ctx.CloseSrcDbIfExists()
	defer ctx.CloseDb()

	for {
		meta, isOk := <-walkChan
		if isOk == false {
			break
		}
		if meta.IsSkipped {
			stats.SkippedOther++
			continue
		}
		if ctx.Limit > 0 && (stats.ProcessedFiles >= ctx.Limit) {
			break
		}
		stats.TotalSize += meta.Size

		DetectDuplication(ctx, &meta)

		if meta.IsDuplicationByChecksum {
			stats.SkippedSameChecksum++
			continue
		}
		if meta.IsDuplicationByDestPath {
			stats.SkippedSameName++
			continue
		}

		Catch(ParseFileExifData(ctx, &meta, ctx.Exiftool))
		//fmt.Println("will setup paths..")
		SetupPaths(ctx, &meta)
		if !ctx.CreateDbOnly {
			CreateDestFile(ctx, meta)
		}

		if ctx.CreateThumbnails && CreateThumbnail(ctx, meta) {
			meta.HasThumbnail = true
		}

		//
		//if ctx.ConvertVideos && meta.IsVideo {
		//	ConvertVideo(ctx, meta)
		//}

		//if stats.ProcessedFiles < 3 {
		//	var jsonmeta, _ = json.Marshal(meta)
		//	fmt.Print(string(jsonmeta))
		//	//fmt.Println(meta.CreationDate)
		//}

		//fmt.Println("will create a DB entry..")
		CreateDbEntry(ctx, meta)

		fileMetaChann <- meta
		stats.ProcessedFiles++
	}

	defer close(fileMetaChann)
}

func SetupPaths(ctx AppContext, meta *FileMeta) {
	meta.Destination = BuildDestination(ctx.DestDir, *meta, DirOriginals)
	meta.Path = filepath.Join(meta.Destination.Dirname, meta.Destination.Basename) + "." + meta.Destination.Extension
	initialOriginPath := FindInitialOriginPath(ctx, *meta)

	if initialOriginPath != meta.OriginPath {
		meta.InitialOriginPath = NullableString(initialOriginPath)
	}
}

func CreateDestFile(ctx AppContext, meta FileMeta) {
	destDir := path.Join(ctx.DestDir, meta.Destination.Dirname)
	destFile := path.Join(destDir, meta.Destination.Basename) + "." + meta.Destination.Extension

	if ctx.DryRun {
		return
	}

	MakeDirIfNotExists(destDir)

	if ctx.MoveFiles {
		Catch(FileMove(meta.Origin.Path, destFile))
	} else {
		Catch(FileCopy(meta.Origin.Path, destFile, true))
	}

	if ctx.FixCreationDates {
		Catch(FileFixDates(destFile, meta.CreationDate, meta.ModificationDate))
	}
}

func CreateDbEntry(ctx AppContext, meta FileMeta) {
	ctx.Db.InsertFileMetaIfNotExists(&meta)
}

func CreateThumbnail(ctx AppContext, meta FileMeta) bool {
	destFile := BuildDestination(ctx.DestDir, meta, DirThumbnails)
	destDir := path.Join(ctx.DestDir, destFile.Dirname)
	MakeDirIfNotExists(destDir)

	var err error = nil

	if regexp.MustCompile("(?i)(^image.*)").MatchString(string(meta.MimeType)) {
		if !regexp.MustCompile(ThumnailableImageMimeTypes).MatchString(string(meta.MimeType)) {
			return false
		}
		if meta.Width > meta.Height {
			err = CreateImageThumbnail(ThumbnailWidth, ThumbnailHeight, imaging.Center, meta.OriginPath, destFile.Path)
		} else {
			err = CreateImageThumbnail(PortraitThumbnailWidth, PortraitThumbnailHeight, imaging.Center, meta.OriginPath, destFile.Path)
		}

		if IsError(err) {
			PrintLnRed("Warning: Cannot create thumbnail image for file " + meta.OriginPath)
			return false
		}
		return true
	}

	if regexp.MustCompile("(?i)(^video.*)").MatchString(string(meta.MimeType)) {
		destFilePath := path.Join(ctx.DestDir, destFile.Dirname, destFile.Basename) + ".gif"
		durationSeconds := time.Duration(meta.Duration) / time.Second
		startTime := int(durationSeconds / 2)
		clipDuration := MinInt(int(durationSeconds), GifMaxDuration)

		if meta.Extension == "3gp" { // fix GIFs having too many seconds
			clipDuration = MinInt(1, clipDuration)
		}

		if meta.Width > meta.Height {
			err = CreateVideoGif(startTime, clipDuration, ThumbnailWidth, GifFrameRate, meta.OriginPath, destFilePath)
		} else {
			err = CreateVideoGif(startTime, clipDuration, PortraitThumbnailWidth, GifFrameRate, meta.OriginPath, destFilePath)
		}

		if IsError(err) {
			PrintLnRed("Warning: Cannot create GIF video for file " + meta.OriginPath)
			return false
		}
		return true
	}

	return false
}

func GetFileChecksum(path string) string {
	checksum, imoErr := imohash.SumFile(path)
	Catch(imoErr)
	return fmt.Sprintf("%x", checksum)
}

func ParseFileExifData(ctx AppContext, meta *FileMeta, exiftool *Exiftool) error {
	exif, err := exiftool.ReadMetadata(meta.OriginPath)
	hasExif := !IsError(err)

	if !hasExif {
		PrintError(err)
	}

	meta.CameraModel = NullableString(exif.GetFullCameraName())
	meta.CreationTool = NullableString(exif.GetFullCreationSoftware())
	meta.IsScreenShot = IsScreenShot(meta.Path, meta.OriginPath, string(meta.CameraModel), string(meta.CreationTool))
	meta.MimeType = NullableString(exif.GetMimeType())
	meta.IsImage = regexp.MustCompile("(?i)(^image.*)").MatchString(string(meta.MimeType))
	meta.IsVideo = regexp.MustCompile("(?i)(^video.*)").MatchString(string(meta.MimeType))
	meta.Width = exif.GetMediaWidth()
	meta.Height = exif.GetMediaHeight()

	err2 := meta.Duration.Parse(exif.GetMediaDuration())
	if IsError(err2) {
		return err2
	}

	gps := exif.GetGPSData()
	meta.GPSAltitude = NullableString(gps.Altitude)
	meta.GPSLatitude = NullableString(ToString(gps.Latitude))
	meta.GPSLongitude = NullableString(ToString(gps.Longitude))
	meta.GPSTimezone = NullableString(gps.Timezone)

	meta.ExifJson = NullableString(exif.DataMapJson)

	meta.CreationDate = DateInTimezone(exif.GetEarliestCreationDate(), gps.Timezone)
	meta.Exif = exif

	return nil
}

func DetectDuplication(ctx AppContext, meta *FileMeta) {
	if PathExists(meta.Path) {
		meta.IsDuplicationByDestPath = true
		return
	}

	if ctx.Db.HasFileMetaByChecksum(meta.Checksum) {
		meta.IsDuplicationByChecksum = true
	}
}

func IsScreenShot(searchStr ...string) bool {
	if regexp.MustCompile(RegexScreenShot).MatchString(strings.Join(searchStr, ":")) {
		return true
	}

	return false
}

func SanitizeExtension(ext string) string {
	ext = strings.Trim(strings.ToLower(ext), ".")

	switch ext {
	case "jpeg":
		return "jpg"
	}
	return ext
}

func BuildDestination(destDirRoot string, data FileMeta, relativeDirName string) FilePathInfo {
	t := data.CreationDate

	ext := SanitizeExtension(data.Origin.Extension)

	var dateFolder, destFilename string

	dateFolder = fmt.Sprintf("%d/%02d", t.Year(), t.Month())

	destFilename = fmt.Sprintf("%d%02d%02d-%02d%02d%02d", t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second()) + "-" + data.Checksum[0:8]

	destDirName := path.Join(relativeDirName, dateFolder)

	return FilePathInfo{
		Basename:  destFilename,
		Dirname:   destDirName,
		Extension: ext,
		Path:      path.Join(destDirRoot, destDirName, destFilename) + "." + ext,
	}
}

func WalkDirRoutine(ctx AppContext) chan FileMeta {
	chann := make(chan FileMeta)
	go func() {
		filepath.Walk(ctx.SrcDir, func(path string, info os.FileInfo, err error) error {
			Catch(err)

			current := FileMeta{
				OriginPath: path,
				IsSkipped:  false,
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
					current.IsSkipped = true
					chann <- current
					return nil
				}
			} else if !regexp.MustCompile(RegexImage).MatchString(path) &&
				!regexp.MustCompile(RegexVideo).MatchString(path) {
				current.IsSkipped = true
				chann <- current
				return nil
			}

			// File is too small?
			if info.Size() < int64(MinFileSize) {
				current.IsSkipped = true
				chann <- current
				return nil
			}

			// Fill basic data
			current.Checksum = GetFileChecksum(path)
			current.Size = info.Size()
			fileExtension := filepath.Ext(path)
			current.Origin = FilePathInfo{
				Path:      path,
				Basename:  strings.Replace(info.Name(), fileExtension, "", -1),
				Dirname:   filepath.Dir(path),
				Extension: fileExtension,
			}
			current.CreationDate = info.ModTime()
			current.ModificationDate = info.ModTime()
			current.Extension = SanitizeExtension(fileExtension)

			chann <- current
			return nil
		})
		defer close(chann)
	}()
	return chann
}

func PrintAppStats(currentFile string, stats AppRunStats, ctx AppContext) {
	PrintReplaceLn(
		"[%s] "+tm.Color(tm.Bold("Stats: %s duplicates + %s skipped | %s processed | %s | %s | %s"), tm.YELLOW),
		AppName,
		ToString(stats.SkippedSameName+stats.SkippedSameChecksum),
		ToString(stats.SkippedOther),
		ToString(stats.ProcessedFiles),
		TotalBytesToString(stats.TotalSize, false),
		time.Since(ctx.StartTime),
		currentFile,
	)
}

func FindExistingExifMetadata(ctx AppContext, file FileMeta) []byte {
	// find in SRC DB
	if ctx.HasSrcMetadataDb() {
		foundFileMeta, found, err := ctx.SrcDb.FindFileMetaByChecksum(file.Checksum)
		Catch(err)
		if found {
			fmt.Print(" // exiftool data found in SRC db")
			return []byte(foundFileMeta.ExifJson)
		}
	}
	// find in DEST DB
	if ctx.HasMetadataDb() {
		foundFileMeta, found, err := ctx.Db.FindFileMetaByChecksum(file.Checksum)
		Catch(err)
		if found {
			fmt.Print(" // exiftool data found in DEST db")
			return []byte(foundFileMeta.ExifJson)
		}
	}

	return nil
}

// get origin path from SRC db instead (if exists)
func FindInitialOriginPath(ctx AppContext, file FileMeta) string {
	if ctx.HasSrcMetadataDb() {
		srcMeta, found, _ := ctx.SrcDb.FindFileMetaByChecksum(file.Checksum)
		if found {
			return srcMeta.OriginPath
		}
		// support old MD5 hashes
		srcMeta, found, _ = ctx.SrcDb.FindFileMetaByChecksum(FileGetMD5Checksum(file.OriginPath))
		if found {
			return srcMeta.OriginPath
		}

		// support legacy DB schemas
		srcMeta2, found2, _ := ctx.SrcDb.LegacyFindFileMetaBy("checksum", FileGetMD5Checksum(file.OriginPath))
		if found2 {
			return srcMeta2.OriginPath
		}
	}
	return file.Origin.Path
}
