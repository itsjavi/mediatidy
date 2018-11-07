package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

func pathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func fileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	defer f.Close()

	if isError(err) {
		return "", err
	}

	h := md5.New()
	if _, err := io.Copy(h, f); isError(err) {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func fileAppend(path, str string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, FilePerms)
	if isError(err) {
		catch(err)
	}

	defer f.Close()

	if _, err = f.WriteString(str); isError(err) {
		catch(err)
	}
}

func fileFixDates(path string, creationDate time.Time, modificationDate time.Time) error {
	if !IsUnix {
		return nil
	}
	err := exec.Command("touch", "-t", creationDate.Format("200601021504.05"), path).Run()

	if isError(err) {
		return err
	}

	err = exec.Command("touch", "-mt", modificationDate.Format("200601021504.05"), path).Run()

	return err
}

func fileCopy(src, dest string, keepAttributes bool) error {
	if keepAttributes == true && IsUnix { // windows does not support cp nor preserving attributes
		err := exec.Command("cp", "-pRP", src, dest).Run()

		return err
	}
	s, err := os.Open(src)
	if isError(err) {
		return err
	}

	defer s.Close()
	d, err := os.Create(dest)
	if isError(err) {
		return err
	}
	if _, err := io.Copy(d, s); isError(err) {
		d.Close()
		return err
	}
	return d.Close()
}

func fileMove(src, dest string) error {
	err := os.Rename(src, dest)

	if isError(err) {
		return err
	}

	return nil
}

func makeDir(dir string) {
	if !pathExists(dir) {
		catch(os.MkdirAll(dir, DirPerms))
	}
}
