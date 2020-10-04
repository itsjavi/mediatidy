package main

const DbSchema = `
CREATE TABLE file (
    checksum TEXT UNIQUE,
    originpath TEXT,
    originname TEXT,
    destinationpath TEXT,
    destinationname TEXT,
    extension TEXT,
    mediatype TEXT,
    size INTEGER,
    creationtime TEXT,
    modificationtime TEXT,
    cameramodel TEXT,
    creationtool TEXT,
    isscreenshot TEXT,
    width TEXT,
    height TEXT,
    duration TEXT,
    gpsaltitude TEXT,
    gpslatitude TEXT,
    gpslongitude TEXT,
    exifjson TEXT
);`

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
	ImageWidth        string
	ImageHeight       string
	Duration          string
	IsScreenShot      bool
	IsDuplication     bool
	IsAlreadyImported bool
	IsLegacyVideo     bool
	Exif              ExifData
	GPS               GPSData
}

//func DbConnection(dbPath string) *sqlx.DB {
//	var db *sqlx.DB
//	db = sqlx.MustConnect("sqlite3", dbPath)
//	db.MustExec(DbSchema)
//	return db
//}
//
//func DbInsertFile(db *sqlx.DB, entity File) {
//	db.MustExec("INSERT INTO place (country, telcode) VALUES ($1, $2)", "Singapore", "65")
//}
//
//func DbFindFile(db *sqlx.DB, checksum string) (File, error) {
//	record := File{}
//	err := db.Get(&record, "SELECT * FROM file WHERE checksum=$1", checksum)
//	if IsError(err) {
//		return record, err
//	}
//	return record, nil
//}
//
//func DbHasFile(db *sqlx.DB, checksum string) bool {
//	_, err := DbFindFile(db, checksum)
//	if !IsError(err) {
//		return false
//	}
//	return true
//}
