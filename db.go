package main

import (
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

type FileMeta struct {
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
	GPSTimezone  string

	ExifJson string `gorm:"type:text"`

	Origin                      FilePathInfo `gorm:"-"`
	Destination                 FilePathInfo `gorm:"-"`
	IsDuplicationByChecksum     bool         `gorm:"-"`
	IsDuplicationByDestBasename bool         `gorm:"-"`
	Exif                        ExifToolData `gorm:"-"`
}

func (FileMeta) TableName() string {
	return "files"
}

type DbHelper struct {
	db        *gorm.DB
	file      string
	connected bool
}

func (dbh *DbHelper) Init(dbFile string, autoMigrate bool) {
	if dbh.connected {
		return
	}

	dbLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Minute,   // Slow SQL threshold
			LogLevel:      logger.Silent, // Log level
			Colorful:      true,          // Disable color
		},
	)
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{Logger: dbLogger})
	nativeDB, nErr := db.DB()

	if err != nil || nErr != nil {
		panic("failed to connect database")
	}

	nativeDB.SetMaxIdleConns(10)
	nativeDB.SetMaxOpenConns(100)
	nativeDB.SetConnMaxIdleTime(time.Hour * 24)

	// Migrate the schema
	if autoMigrate {
		db.AutoMigrate(&FileMeta{})
	}

	dbh.db = db
	dbh.file = dbFile
	dbh.connected = true
}

func (dbh *DbHelper) InsertFileMeta(fileMeta *FileMeta) error {
	return dbh.db.Create(&fileMeta).Error
}

func (dbh *DbHelper) FindFileMetaByChecksum(checksum string) (FileMeta, bool, error) {
	return dbh.FindFileMetaBy("checksum", checksum)
}

func (dbh *DbHelper) HasFileMetaByChecksum(checksum string) bool {
	return dbh.HasFileMetaBy("checksum", checksum)
}

func (dbh *DbHelper) FindFileMetaBy(column string, val string) (FileMeta, bool, error) {
	var record = FileMeta{}
	result := dbh.db.First(&record, column+" = ?", val)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return record, false, nil
	}
	return record, record.ID > 0, result.Error
}

func (dbh *DbHelper) HasFileMetaBy(column string, val string) bool {
	_, found, err := dbh.FindFileMetaBy(column, val)
	HandleError(err)
	return found
}

func (dbh *DbHelper) InsertFileMetaIfNotExists(file *FileMeta) bool {
	if dbh.HasFileMetaByChecksum(file.Checksum) {
		return false
	}

	HandleError(dbh.InsertFileMeta(file))

	return file.ID > 0
}
