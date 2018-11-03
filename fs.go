package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
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

	if err != nil {
		return "", err
	}

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func fileAppend(path, str string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, PathPerms)
	if err != nil {
		catch(err)
	}

	defer f.Close()

	if _, err = f.WriteString(str); err != nil {
		catch(err)
	}
}

func fileCopy(src, dest string, keepAttributes bool) error {
	if keepAttributes == true && runtime.GOOS != "windows" { // windows does not support cp nor preserving attributes
		_, err := exec.Command("cp", "-pRP", src, dest).Output()

		return err
	}
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

func fileMove(src, dest string) error {
	err := os.Rename(src, dest)

	if err != nil {
		return err
	}

	return nil
}

func makeDir(dir string) {
	if !pathExists(dir) {
		catch(os.MkdirAll(dir, PathPerms))
	}
}
