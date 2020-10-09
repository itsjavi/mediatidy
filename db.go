package main

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"regexp"
	"time"
)

type NullableString string

func (NullableString) GormDataType() string {
	return "text"
}

// Scan scan value into int64, implements sql.Scanner interface
func (s *NullableString) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return errors.New(fmt.Sprint("Failed to convert value to string:", value))
	}

	*s = NullableString(str)

	return nil
}

// Value return json value, implement driver.Valuer interface
func (s NullableString) Value() (driver.Value, error) {
	if s == "" {
		return nil, nil
	}
	return string(s), nil
}

type MediaDuration time.Duration

func (MediaDuration) GormDataType() string {
	return "string"
}

// Scan scan value into int64, implements sql.Scanner interface
func (d *MediaDuration) Scan(value interface{}) error {
	formattedDuration, ok := value.(string)
	if !ok {
		return errors.New(fmt.Sprint("Failed to convert value to string:", value))
	}

	return d.Parse(formattedDuration)
}

// Value return json value, implement driver.Valuer interface
func (d MediaDuration) Value() (driver.Value, error) {
	if d == 0 {
		return nil, nil
	}
	return fmt.Sprintf("%s", time.Duration(d)), nil
}

// Parse a media duration value.
// Compatible formats are: 1.23s, 1.23s (approx), 00:00:01, 00:00:01 (approx), 00:00:01.23, 00:00:01.23 (approx)
func (d *MediaDuration) Parse(value string) error {
	if value == "" {
		*d = 0
		return nil
	}
	hmsRegex := regexp.MustCompile("(?i)(^[0-9]{1,}):([0-9]{1,}):([0-9.]{1,})$")
	if hmsRegex.MatchString(value) {
		value = hmsRegex.ReplaceAllString(value, "${1}h${2}m${3}s")
	}

	nanoSeconds, err := time.ParseDuration(value)
	if IsError(err) {
		return err
	}

	*d = MediaDuration(nanoSeconds)
	return nil
}

type LegacyFileMeta struct {
	ID         uint   `gorm:"primarykey"`
	Checksum   string `gorm:"type:string;size:32;uniqueIndex"`
	Path       string `gorm:"type:text;index"`
	OriginPath string `gorm:"type:text;index"`
}

func (LegacyFileMeta) TableName() string {
	return "files"
}

type FileMeta struct {
	ID uint `gorm:"primarykey"`

	// File
	Checksum          string `gorm:"type:text;uniqueIndex"`
	Size              int64
	Path              string         `gorm:"type:text;index"`
	OriginPath        string         `gorm:"type:text;index"`
	InitialOriginPath NullableString `gorm:"type:text;index"`
	Extension         string
	CreationDate      time.Time
	ModificationDate  time.Time

	// Basic metadata
	MimeType     NullableString
	Width        int
	Height       int
	Duration     MediaDuration
	IsImage      bool
	IsVideo      bool
	IsScreenShot bool

	// thumbnail
	ThumbnailPath   NullableString
	ThumbnailWidth  int
	ThumbnailHeight int

	// Extra metadata
	CameraModel  NullableString
	CreationTool NullableString
	GPSAltitude  NullableString
	GPSLatitude  NullableString
	GPSLongitude NullableString
	GPSTimezone  NullableString
	ExifJson     NullableString

	// Internal
	Exif                    ExifToolMetadata `gorm:"-"`
	Origin                  FilePathInfo     `gorm:"-"`
	Destination             FilePathInfo     `gorm:"-"`
	IsSkipped               bool             `gorm:"-"`
	IsDuplicationByChecksum bool             `gorm:"-"`
	IsDuplicationByDestPath bool             `gorm:"-"`
}

type FilePathInfo struct {
	Path      string
	Basename  string
	Dirname   string
	Extension string
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

func (dbh *DbHelper) Close() error {
	dbc, err := dbh.db.DB()
	if IsError(err) {
		return err
	}
	return dbc.Close()
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

func (dbh *DbHelper) LegacyFindFileMetaBy(column string, val string) (LegacyFileMeta, bool, error) {
	var record = LegacyFileMeta{}
	result := dbh.db.First(&record, column+" = ?", val)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return record, false, nil
	}
	return record, record.ID > 0, result.Error
}

func (dbh *DbHelper) HasFileMetaBy(column string, val string) bool {
	_, found, err := dbh.FindFileMetaBy(column, val)
	Catch(err)
	return found
}

func (dbh *DbHelper) InsertFileMetaIfNotExists(file *FileMeta) bool {
	if dbh.HasFileMetaByChecksum(file.Checksum) {
		return false
	}

	Catch(dbh.InsertFileMeta(file))

	return file.ID > 0
}
