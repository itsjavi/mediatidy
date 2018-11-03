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
	if *params.move {
		action = "[MV]"
	}

	err = filepath.Walk(params.src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Print which path is being analysed
		if info.IsDir() && !strings.ContainsAny(path, "#@") {
			scannedDir := strings.Replace(path, params.src+"/", "", -1)
			if len(scannedDir) > 100 {
				scannedDir = scannedDir[0:99] + "..."
			}
			logSameLn(">> %s Analyzing: %s", action, scannedDir)
		}

		if info.IsDir() {
			return nil
		}

		if regexp.MustCompile(RegexExcludeDirs).MatchString(path) {
			params.total.skipped++
			return nil
		}

		processed := params.total.unique + params.total.duplicated

		if (*params.limit > 0) && (processed >= *params.limit) {
			return LimitExceededError
		}

		relPath := strings.Replace(path, params.src+"/", "", -1)
		if len(relPath) > 100 {
			relPath = relPath[0:99] + "..."
		}

		logSameLn(">> %s Parsing: %s", action, relPath)

		data, err := buildFileData(params, path, info)

		if err != nil {
			return err
		}

		if data.flags.skipped {
			params.total.skipped++
			return nil
		}

		// logSameLn(">> Storing: %s", relPath)

		err = storeFile(data, params)
		if err != nil {
			return err
		}

		if data.flags.unique {
			params.total.unique++
		}

		if data.flags.duplicated {
			params.total.duplicated++
		}

		return nil
	})

	fmt.Printf("\n\n")

	if err != LimitExceededError {
		catch(err)
	}

	if !*params.dryRun {
		writeLogFile(params)
	}

	writeLogOutput(params)
}
