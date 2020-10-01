package main

import (
	"os"
)

func DirExists(dir string) bool {
	dirStat, err := os.Stat(dir)

	if os.IsNotExist(err) {
		return false
	}

	return dirStat.IsDir()
}
