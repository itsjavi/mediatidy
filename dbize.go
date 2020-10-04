package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/buger/goterm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type File struct {
	ID         uint   `gorm:"primarykey"`
	Checksum   string `gorm:"type:string;size:32;uniqueIndex"`
	Path       string `gorm:"type:text;index"`
	OriginPath string `gorm:"type:text;index"`
	Extension  string `gorm:"type:string;size:32"`

	MediaType string `gorm:"type:string;size:32"`
	Size      int64

	CreationDate     string
	ModificationDate string

	CameraModel  string
	CreationTool string
	IsScreenShot bool

	Width    string
	Height   string
	Duration string

	GPSAltitude  string
	GPSLatitude  string
	GPSLongitude string

	ExifJson string `gorm:"type:text"`
}

func InitDb(path string) *gorm.DB {
	dbLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Minute,   // Slow SQL threshold
			LogLevel:      logger.Silent, // Log level
			Colorful:      true,          // Disable color
		},
	)
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: dbLogger})

	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&File{})

	return db
}

func DBize(metadataDir string) {
	db := InitDb(filepath.Join(metadataDir, "metadata.sqlite"))
	fmt.Println("Migrating metadata JSON files to SQLite...")

	filepath.Walk(metadataDir, func(path string, info os.FileInfo, err error) error {
		if IsError(err) {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !regexp.MustCompile("(?i)\\.(json)$").MatchString(path) {
			return nil
		}

		// Load metadata file and find it in DB
		meta, err := loadMetadataJson(path)
		HandleError(err)
		var foundFile = File{}
		result := db.First(&foundFile, "checksum = ?", meta.Checksum)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Insert new entry
			newFile := transformFileMeta(meta)
			result = db.Create(&newFile)
			HandleError(result.Error)

			PrintReplaceLn(" -> %s", newFile.Path)
			if !PathExists(filepath.Join(filepath.Dir(metadataDir), newFile.Path)) {
				fmt.Println(goterm.Color("Original not found: "+newFile.Path, goterm.RED))
				fmt.Println("")
			}
		} else {
			HandleError(result.Error)
		}

		return nil
	})
}

func transformFileMeta(meta FileMeta) File {
	return File{
		Checksum:         meta.Checksum,
		Path:             filepath.Join(meta.Destination.Dirname, meta.Destination.Basename) + meta.Destination.Extension,
		OriginPath:       meta.Source.Path,
		Extension:        strings.TrimLeft(meta.Destination.Extension, "."),
		MediaType:        meta.MediaType,
		Size:             meta.Size,
		CreationDate:     meta.CreationTime,
		ModificationDate: meta.ModificationTime,
		CameraModel:      meta.CameraModel,
		CreationTool:     meta.CreationTool,
		IsScreenShot:     meta.IsScreenShot,
		Width:            meta.Exif.Data.ImageWidth,
		Height:           meta.Exif.Data.ImageHeight,
		Duration:         meta.Duration,
		GPSAltitude:      meta.GPS.Position.Altitude,
		GPSLatitude:      ToString(meta.GPS.Position.Latitude),
		GPSLongitude:     ToString(meta.GPS.Position.Longitude),
		ExifJson:         meta.Exif.DataDumpRaw,
	}
}

func loadMetadataJson(metaFile string) (FileMeta, error) {
	var meta FileMeta
	metadataBytes, err := ioutil.ReadFile(metaFile)
	if !IsError(err) {
		jsonerr := json.Unmarshal(metadataBytes, &meta)
		HandleError(jsonerr)

		exifData := ParseExifMetadata([]byte(meta.Exif.DataDumpRaw))
		meta.GPS = GPSDataParse(exifData.GPSPosition, exifData.GPSAltitude)
		meta.Duration = ParseMediaDuration(exifData)
		meta.Exif.Data = exifData

		return meta, nil
	}
	return meta, err
}
