package main

import (
	"fmt"
	tm "github.com/buger/goterm"
	"github.com/kalafut/imohash"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
		SetupPaths(ctx, &meta)
		if !ctx.CreateDbOnly {
			CreateDestFile(ctx, meta)
		}
		CreateDbEntry(ctx, meta)

		//if ctx.CreateThumbnails {
		//	CreateThumbnail(ctx, meta)
		//}
		//
		//if ctx.ConvertVideos && meta.IsVideo {
		//	ConvertVideo(ctx, meta)
		//}

		//if stats.ProcessedFiles < 3 {
		//	var jsonmeta, _ = json.Marshal(meta)
		//	fmt.Print(string(jsonmeta))
		//	//fmt.Println(meta.CreationDate)
		//}

		fileMetaChann <- meta
		stats.ProcessedFiles++
	}

	defer close(fileMetaChann)
}

func SetupPaths(ctx AppContext, meta *FileMeta) {
	meta.Destination = BuildDestination(ctx.DestDir, *meta)
	meta.Path = filepath.Join(meta.Destination.Dirname, meta.Destination.Basename) + "." + meta.Destination.Extension
	initialOriginPath := FindInitialOriginPath(ctx, *meta)

	if initialOriginPath != meta.OriginPath {
		meta.InitialOriginPath = NullableString(initialOriginPath)
	}
}

func CreateDestFile(ctx AppContext, meta FileMeta) {
	destDir := ctx.DestDir + "/" + meta.Destination.Dirname

	if ctx.DryRun {
		return
	}

	MakeDirIfNotExists(destDir)

	if ctx.MoveFiles {
		Catch(FileMove(meta.Origin.Path, meta.Path))
	} else {
		Catch(FileCopy(meta.Origin.Path, meta.Path, true))
	}

	if ctx.FixCreationDates {
		Catch(FileFixDates(meta.Path, meta.CreationDate, meta.ModificationDate))
	}
}

func CreateDbEntry(ctx AppContext, meta FileMeta) {
	ctx.Db.InsertFileMetaIfNotExists(&meta)
}

func CreateThumbnail(ctx AppContext, meta FileMeta) {

}

func ConvertVideo(ctx AppContext, meta FileMeta) {

}

func GetFileChecksum(path string) string {
	checksum, imoErr := imohash.SumFile(path)
	Catch(imoErr)
	return fmt.Sprintf("%x", checksum)
}

func ParseFileExifData(ctx AppContext, meta *FileMeta, exiftool *Exiftool) error {
	exif, err := exiftool.ReadMetadata(meta.OriginPath)

	if IsError(err) {
		return err
	}

	meta.CameraModel = NullableString(exif.GetFullCameraName())
	meta.CreationTool = NullableString(exif.GetFullCreationSoftware())
	meta.IsScreenShot = IsScreenShot(meta.Path, meta.OriginPath, string(meta.CameraModel), string(meta.CreationTool))

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
	meta.MimeType = NullableString(exif.GetMimeType())
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

func BuildDestination(destDirRoot string, data FileMeta) FilePathInfo {
	t := data.CreationDate

	ext := SanitizeExtension(data.Origin.Extension)

	var dateFolder, destFilename string

	dateFolder = fmt.Sprintf("%d/%02d", t.Year(), t.Month())

	destFilename = fmt.Sprintf("%d%02d%02d-%02d%02d%02d", t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second()) + "-" + data.Checksum[0:8]

	destDirName := path.Join(DirOriginals + dateFolder)

	return FilePathInfo{
		Basename:  destFilename,
		Dirname:   destDirName,
		Extension: ext,
		Path:      destDirRoot + "/" + destDirName + "/" + destFilename + "." + ext,
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
		"[%s] "+tm.Color(tm.Bold("Stats: %s duplicates | %s skipped | %s processed | %s total size | %s elapsed time | file: %s"), tm.YELLOW),
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
		srcMeta, found, err := ctx.SrcDb.FindFileMetaByChecksum(file.Checksum)
		Catch(err)
		if found {
			return srcMeta.OriginPath
		}
		// support old MD5 hashes
		srcMeta, found, err = ctx.SrcDb.FindFileMetaByChecksum(FileGetMD5Checksum(file.OriginPath))
		Catch(err)
		if found {
			return srcMeta.OriginPath
		}
	}
	return file.Origin.Path
}
