package main

import (
	"fmt"
	"github.com/kalafut/imohash"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func TidyRoutine(ctx AppContext, stats *WalkDirStats, fileMetaChann chan FileMeta) {
	walkChan := WalkDirRoutine(ctx)
	exifTool := Exiftool{}
	exifTool.UseDefaults()
	Catch(exifTool.Open())
	defer exifTool.Close()

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

		Catch(
			DetectDuplication(ctx, &meta),
		)

		if meta.IsDuplicationByChecksum {
			stats.SkippedSameChecksum++
			continue
		}

		if meta.IsDuplicationByDestBasename {
			stats.SkippedSameName++
			continue
		}

		Catch(
			ParseFileExifData(ctx, &meta, &exifTool),
		)

		//fmt.Println(meta.CreationDate, meta.Exif.Get("CreateDate"))

		meta.Destination = BuildDestination(ctx.DestDir, meta)
		meta.Path = filepath.Join(meta.Destination.Dirname, meta.Destination.Basename) + meta.Destination.Extension

		CreateDestFile(ctx, meta)
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

func CreateDestFile(ctx AppContext, meta FileMeta) {
	destDir := ctx.DestDir + "/" + meta.Destination.Dirname
	destFile := destDir + "/" + meta.Destination.Basename + meta.Destination.Extension

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
		ct, err := ParseDateWithTimezone(time.RFC3339, meta.CreationDate, meta.GPSTimezone)
		mt, err2 := ParseDateWithTimezone(time.RFC3339, meta.ModificationDate, meta.GPSTimezone)

		if !IsError(err) && !IsError(err2) {
			Catch(FileFixDates(destFile, ct, mt))
		}
	}
}

func CreateDbEntry(ctx AppContext, meta FileMeta) {

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

	meta.CameraModel = exif.GetFullCameraName()
	meta.CreationTool = exif.GetFullCreationSoftware()
	meta.IsScreenShot = IsScreenShot(meta.Path, meta.OriginPath, meta.CameraModel, meta.CreationTool)

	meta.Width = ToString(exif.GetMediaWidth())
	meta.Height = ToString(exif.GetMediaHeight())
	meta.Duration = exif.GetMediaDuration()

	gps := exif.GetGPSData()
	meta.GPSAltitude = gps.Altitude
	meta.GPSLatitude = ToString(gps.Latitude)
	meta.GPSLongitude = ToString(gps.Longitude)
	meta.GPSTimezone = gps.Timezone

	meta.ExifJson = exif.DataMapJson

	meta.CreationDate = FormatDateWithTimezone(exif.GetEarliestCreationDate(), gps.Timezone)
	meta.MediaType = exif.GetMimeType()
	meta.Exif = exif

	return nil
}

func DetectDuplication(ctx AppContext, meta *FileMeta) error {
	return nil
}

func IsScreenShot(searchStr ...string) bool {
	if regexp.MustCompile(RegexScreenShot).MatchString(strings.Join(searchStr, ":")) {
		return true
	}

	return false
}

func SanitizeExtension(ext string) string {
	ext = strings.ToLower(ext)

	switch ext {
	case "jpeg":
		return "jpg"
	}
	return ext
}

func BuildDestination(destDirRoot string, data FileMeta) FilePathInfo {
	t, err := time.Parse(time.RFC3339, data.CreationDate)
	Catch(err)

	ext := SanitizeExtension(data.Origin.Extension)

	var dateFolder, destFilename string

	dateFolder = fmt.Sprintf("%d/%02d", t.Year(), t.Month())

	destFilename = fmt.Sprintf("%d%02d%02d-%02d%02d%02d", t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second()) + "-" + data.Checksum[0:8]

	destDirName := "originals/" + dateFolder

	return FilePathInfo{
		Basename:  destFilename,
		Dirname:   destDirName,
		Extension: ext,
		Path:      destDirRoot + "/" + destDirName + "/" + destFilename + ext,
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
			fileExtension := strings.ToLower(filepath.Ext(path))
			current.Origin = FilePathInfo{
				Path:      path,
				Basename:  strings.Replace(info.Name(), fileExtension, "", -1),
				Dirname:   filepath.Dir(path),
				Extension: fileExtension,
			}
			current.CreationDate = info.ModTime().Format(DateLayout)
			current.ModificationDate = info.ModTime().Format(DateLayout)
			current.Extension = SanitizeExtension(fileExtension)

			chann <- current
			return nil
		})
		defer close(chann)
	}()
	return chann
}
