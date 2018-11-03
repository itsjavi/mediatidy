package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
		if err != nil {
			return err
		}

		// Print which path is being analysed
		if info.IsDir() && !strings.ContainsAny(path, "#@") {
			scannedDir := strings.Replace(path, params.Src+"/", "", -1)
			if len(scannedDir) > 100 {
				scannedDir = scannedDir[0:99] + "..."
			}
			logSameLn(">> %s Analyzing: %s", action, scannedDir)
		}

		if info.IsDir() {
			return nil
		}

		if regexp.MustCompile(RegexExcludeDirs).MatchString(path) {
			params.Total.Skipped++
			return nil
		}

		processed := params.Total.Unique + params.Total.Duplicated

		if (*params.Limit > 0) && (processed >= *params.Limit) {
			return LimitExceededError
		}

		relPath := strings.Replace(path, params.Src+"/", "", -1)
		if len(relPath) > 80 {
			relPath = relPath[0:79] + "..."
		}

		logSameLn(">> %s Processing: %s (%d / %d d / %d s)",
			action, relPath, params.Total.Unique, params.Total.Duplicated, params.Total.Skipped)

		data, err := buildFileData(params, path, info)

		if err != nil {
			return err
		}

		if data.flags.skipped {
			params.Total.Skipped++
			return nil
		}

		// logSameLn(">> Storing: %s", relPath)

		err = storeFile(data, params)
		if err != nil {
			return err
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
