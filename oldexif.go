package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)



func GetFileMetadata(ctx AppContext, path string, info os.FileInfo) (FileMeta, error) {
	fileExtension := strings.ToLower(filepath.Ext(path))

	fileMeta := FileMeta{
		Origin: FilePathInfo{
			Path:      path,
			Basename:  strings.Replace(info.Name(), fileExtension, "", -1),
			Dirname:   filepath.Dir(path),
			Extension: fileExtension,
		},
		Size:                        info.Size(),
		Checksum:                    FileCalcChecksum(path),
		MediaType:                   "",
		IsDuplicationByChecksum:     false,
		IsDuplicationByDestBasename: false,
	}

	// Build Destination file name and dirName
	fileMeta.Destination = BuildDestination(ctx.DestDir, fileMeta)
	alreadyExists := PathExists(fileMeta.Destination.Path) || ctx.Db.HasFileMetaByChecksum(fileMeta.Checksum)

	if alreadyExists {
		// Detect duplication by checksum or Destination path (e.g. when trying to copy twice from same folder)
		if filepath.Base(path) == filepath.Base(fileMeta.Destination.Path) {
			// skip storing duplicate if same filename
			fileMeta.IsDuplicationByDestBasename = true
			return fileMeta, nil
		}
		fileMeta.IsDuplicationByChecksum = true
		return fileMeta, nil
	}

	return fileMeta, nil
}

func readExifMetadata(ctx AppContext, file FileMeta) []byte {
	// find in SRC DB
	if ctx.HasSrcMetadataDb() {
		foundFileMeta, found, err := ctx.SrcDb.FindFileMetaByChecksum(file.Checksum)
		if found {
			fmt.Print(" // exiftool data found in SRC db")
			return []byte(foundFileMeta.ExifJson)
		}
		Catch(err)
	}
	// find in DEST DB
	if ctx.HasMetadataDb() {
		foundFileMeta, found, err := ctx.Db.FindFileMetaByChecksum(file.Checksum)
		if found {
			fmt.Print(" // exiftool data found in DEST db")
			return []byte(foundFileMeta.ExifJson)
		}
		Catch(err)
	}

	fallbackMetadata := []byte(`[{"SourceFile":"` + file.Origin.Path + `", "Error": true}]`)

	jsonBytes, err := exec.Command("exiftool", file.Origin.Path, "-json", "-api", "largefilesupport=1", "-extractEmbedded").Output()
	if IsError(err) {
		return fallbackMetadata
	}
	fmt.Print(" // exiftool data extracted from file")

	return jsonBytes
}
