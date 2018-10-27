package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gosimple/slug"
	"github.com/xiam/exif"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	AppName     = "convergent"
	MinFileSize = 10000 // 15 kB
	PathPerms   = 0755
)

func catch(e error) {
	if e != nil {
		log.Fatal("Unexpected error: " + e.Error())
		panic(e)
	}
}

func PathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func fsize(path string) int64 {
	fi, err := os.Stat(path)
	catch(err)
	return fi.Size()
}

func fappend(path, str string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, PathPerms)
	if err != nil {
		catch(err)
	}

	defer f.Close()

	if _, err = f.WriteString(str); err != nil {
		catch(err)
	}
}

func LogLn(message string, a ...interface{}) {
	fmt.Printf("["+AppName+"] "+message+"\n", a...)
}

func hashFileMd5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil

}

func getCameraAndDate(path string) (string, string) {
	camera := "other"

	if regexp.MustCompile(`(?m)(?i)(Screen Shot|Captura|Screenshot)`).MatchString(path) {
		camera = "screenshots"
	} else if regexp.MustCompile(`(?m)(?i)(facebook)`).MatchString(path) {
		camera = "facebook"
	} else if regexp.MustCompile(`(?m)(?i)(instagram)`).MatchString(path) {
		camera = "instagram"
	} else if regexp.MustCompile(`(?m)(?i)(twitter)`).MatchString(path) {
		camera = "twitter"
	} else if regexp.MustCompile(`(?m)(?i)(whatsapp)`).MatchString(path) {
		camera = "whatsapp"
	} else if regexp.MustCompile(`(?m)(?i)(telegram)`).MatchString(path) {
		camera = "telegram"
	} else if regexp.MustCompile(`(?m)(?i)(messenger)`).MatchString(path) {
		camera = "messenger"
	} else if regexp.MustCompile(`(?m)(?i)(snapchat)`).MatchString(path) {
		camera = "snapchat"
	}

	fallbackCamera := camera

	fileInfo, err := os.Stat(path)
	if err != nil {
		return camera, ""
	}
	dateOriginalFallback := fileInfo.ModTime().Format("2006:01:02 15:04:05") // fallback

	data, err := exif.Read(path)
	if err != nil {
		return camera, dateOriginalFallback
	}

	camera = slug.Make(strings.Trim(fmt.Sprintf("%s %s", data.Tags["Manufacturer"], data.Tags["Model"]), " "))
	dateOriginal := data.Tags["Date and Time (Original)"]

	if dateOriginal == "" {
		dateOriginal = data.Tags["Date and Time (Digitized)"]
	}

	if dateOriginal == "" {
		dateOriginal = dateOriginalFallback
	}

	if camera == "" {
		camera = fallbackCamera
	}

	return camera, dateOriginal
}

func getNewPath(path string, camera string, date string, mediaType string) (string, string) {
	t, err := time.Parse("2006:01:02 15:04:05", date)
	dateFolder := "2000/" + camera + "/01"
	dateName := "2000-01-01-00-00-00"

	if err == nil {
		// dateFolder = fmt.Sprintf("%s/%d/%02d/%02d", camera, t.Year(), t.Month(), t.Day())
		if mediaType != "images" && camera == "other" {
			dateFolder = fmt.Sprintf("%d/%02d", t.Year(), t.Month())
		} else {
			dateFolder = fmt.Sprintf("%d/%s/%02d", t.Year(), camera, t.Month())
		}
		dateName = fmt.Sprintf("%d%02d%02d-%02d%02d%02d", t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second())
	}

	fileMd5, _ := hashFileMd5(path)
	if fileMd5 == "" {
		panic("Cannot create MD5 for file: " + path)
		//hash := md5.New();
		//hash.Write([]byte(uuid.NewV4().String() + path + dateFolder + dateName + mediaType))
		//fileMd5 := hex.EncodeToString(hash.Sum(nil)[:16])
	}

	fileName := dateName + "-" + fileMd5 + strings.ToLower(filepath.Ext(path))
	return mediaType + "/" + dateFolder, fileName
}

func cp(src, dest string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}

	defer s.Close()
	d, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

func mv(src, dest string) error {
	err := os.Rename(src, dest)

	if err != nil {
		return err
	}

	return nil
}

func mkdirp(dir string) {
	if !PathExists(dir) {
		catch(os.MkdirAll(dir, PathPerms))
	}
}

type FileMetadata map[string]interface{}

func extractMetadataJson(path string) []byte {
	out, err := exec.Command("exiftool", path, "-json").Output()
	if err != nil {
		return []byte(`[{"SourceFile":"` + path + `"}]`)
	}

	return out
}

func extractMetadata(path string) FileMetadata {
	var data []FileMetadata
	jsonData := extractMetadataJson(path)
	catch(json.Unmarshal([]byte(jsonData), &data))

	return data[0]
}

func processFile(path string, fileName string, destRoot string, destDir string, moveFiles bool) bool {
	absDestDir := destRoot + "/" + destDir
	absDestMetaDir := destRoot + "/.metadata/" + destDir
	absDestDuplDir := destRoot + "/.duplicates/" + destDir
	absDestDuplMetaDir := destRoot + "/.duplicates/.metadata/" + destDir
	absDestFile := absDestDir + "/" + fileName
	metaFileName := strings.Replace(fileName, strings.ToLower(filepath.Ext(path)), "", -1) + ".json"
	absDestMetaFile := absDestMetaDir + "/" + metaFileName

	jsonData := extractMetadataJson(path)
	isDuplicated := false

	mkdirp(absDestDir)
	mkdirp(absDestMetaDir)

	if PathExists(absDestFile) {
		isDuplicated = true

		mkdirp(absDestDuplDir)
		mkdirp(absDestDuplMetaDir)

		//if !PathExists(absDestDuplDir + "/" + fileName) {
		if moveFiles {
			catch(mv(path, absDestDuplDir+"/"+fileName))
		} else {
			catch(cp(path, absDestDuplDir+"/"+fileName))
		}
		//}

		//if !PathExists(absDestDuplMetaDir + "/" + metaFileName) {
		catch(ioutil.WriteFile(absDestDuplMetaDir+"/"+metaFileName, jsonData, PathPerms))
		//}
	} else {
		if moveFiles {
			catch(mv(path, absDestFile))
		} else {
			catch(cp(path, absDestFile))
		}
	}

	if !PathExists(absDestMetaFile) {
		catch(ioutil.WriteFile(absDestMetaDir+"/"+metaFileName, jsonData, PathPerms))
	}

	return isDuplicated
}

func tpl(str string, vars ...interface{}) string {
	return fmt.Sprintf(str, vars...)
}

func main() {
	limit := flag.Int("limit", 0, "Limit the amount of processed files")
	moveFiles := flag.Bool("move", false, "Moves the files instead of copying them")

	flag.Parse()

	src := flag.Arg(0)
	dest := flag.Arg(1)

	imageReg := regexp.MustCompile("(?i)\\.(jpg|jpeg|gif|png|webp|tiff|bmp|raw)$")
	//imageJpegReg := regexp.MustCompile("(?i)\\.(jpg|jpeg)$")
	videoReg := regexp.MustCompile("(?i)\\.(mpg|wmv|avi|mov|m4v|3gp|mp4|flv|webm|ogv|ts)$")
	audioReg := regexp.MustCompile("(?i)\\.(mp3|m4a|aac|wav|ogg|oga|wma|flac)$")

	if src == "" {
		panic("Missing argument: source-folder")
	}

	if dest == "" {
		dest = src + "-" + AppName
	}

	found := 0
	duplicates := 0
	skipped := 0

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		mediaType := ""

		if imageReg.MatchString(path) {
			mediaType = "images"
		}

		if videoReg.MatchString(path) {
			mediaType = "videos"
		}

		if audioReg.MatchString(path) {
			mediaType = "audios"
		}

		if mediaType == "" {
			return nil
		}

		if (*limit > 0) && ((found + duplicates) >= *limit) {
			return nil
		}

		fileSize := fsize(path)

		if fileSize < 1000 { // < 1 KB
			skipped++
			LogLn("   (skipping too small file)  " + strings.Replace(path, src, "", -1))
			return nil
		}

		camera, date := getCameraAndDate(path)
		newDir, newFilename := getNewPath(path, camera, date, mediaType)

		if fileSize < MinFileSize {
			newDir = ".small/" + newDir
		}

		isDuplicated := processFile(path, newFilename, dest, newDir, *moveFiles)

		if isDuplicated {
			duplicates++
			LogLn("   (duplicated)  " + newDir + "/" + newFilename)
		} else {
			found ++
			LogLn(strconv.Itoa(found) + " - " + newDir + "/" + newFilename)
		}

		return nil
	})

	catch(err)

	log := tpl(
		`{"srcRoot":"%s", "destRoot":"%s", "unique": %d, "duplicated": %d, "skipped": %d, "date": "%s"}`,
		src, dest, found, duplicates, skipped, time.Now().Format(time.RFC3339))

	fappend(dest+"/"+AppName+".log", log+"\n")

	LogLn("%d media files (+%d duplicates) found under `%s`", found, duplicates, src)

	if *moveFiles {
		LogLn("%d files moved, %d skipped", found+duplicates, skipped)
	} else {
		LogLn("%d files copied, %d skipped", found+duplicates, skipped)
	}
}
