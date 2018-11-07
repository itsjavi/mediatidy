package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

/*
PHASES:
1. Parse CLI arguments
2. Walk path
3. Check if path is file and readable
4. Check if file is supported type
5. Check if file is not too small
6. Extract Metadata
7. Build destination path name (media type, camera, date)
8. Build destination filename (date/slug, file md5)
9. Detect if absolute file destination path is a duplicate
10. Move/copy file
11. Create metadata JSON file
12. Add entry into log file
 */

func main() {
	params, err := createAppParams()
	catch(err)

	fmt.Printf("\n")
	action := "[CP]"
	if params.move {
		action = "[MV]"
	}

	err = filepath.Walk(params.Src, func(path string, info os.FileInfo, err error) error {
		if isError(err) {
			return err
		}

		// Skip development-related sibling directories
		if info.IsDir() && (pathExists(filepath.Dir(path)+"/.git") || pathExists(filepath.Dir(path)+"/.idea")) {
			return filepath.SkipDir
		}

		if regexp.MustCompile(RegexExcludeDirs).MatchString(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			params.Total.Skipped++
			return nil
		}

		if info.IsDir() {
			return nil
		}

		processed := params.Total.Unique + params.Total.Duplicated

		if (*params.Limit > 0) && (processed > *params.Limit) {
			return LimitExceededError
		}

		pathShort := ".../" + filepath.Base(filepath.Dir(path)) + "/" + filepath.Base(path)
		if len(pathShort) > 50 {
			pathShort = pathShort[0:49] + "..."
		}

		logSameLn(">> [%s] %12v %8v %s %8v GPS %8v dup %8v skip  |  curr: %s",
			AppName,
			ByteCountToHumanReadable(params.Total.Size, false),
			params.Total.Unique,
			action,
			params.Total.WithGPS,
			params.Total.Duplicated, params.Total.Skipped,
			pathShort,
		)

		data, err := buildFileData(params, path, info)

		if isError(err) {
			return err
		}

		if data.flags.skipped {
			params.Total.Skipped++
			return nil
		}

		err = storeFile(data, params)
		if isError(err) {
			return err
		}

		params.Total.Size += data.Size
		if data.GPSTimezone != "" {
			params.Total.WithGPS++
		}

		if data.flags.unique {
			params.Total.Unique++
		}

		if data.flags.duplicated {
			params.Total.Duplicated++
		}

		return nil
	})

	fmt.Printf("\n\n")

	if err != LimitExceededError {
		catch(err)
	}

	if !*params.dryRun && (params.Total.Unique > 0) {
		writeLogFile(params)
	}

	writeLogOutput(params)
}
