package fs

import (
	"crypto/md5"
	"fmt"
	"github.com/itsjavi/happytimes/utils"
	"io"
	"os"
	"os/exec"
	"time"
)

const (
	DirPerms  = 0755
	FilePerms = 0644
)

func PathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func FileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	defer f.Close()

	if utils.IsError(err) {
		return "", err
	}

	h := md5.New()
	if _, err := io.Copy(h, f); utils.IsError(err) {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func FileAppend(path, str string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, FilePerms)
	if utils.IsError(err) {
		utils.Catch(err)
	}

	defer f.Close()

	if _, err = f.WriteString(str); utils.IsError(err) {
		utils.Catch(err)
	}
}

func FileFixDates(path string, creationDate time.Time, modificationDate time.Time) error {
	if !utils.IsUnix {
		return nil
	}
	err := exec.Command("touch", "-t", creationDate.Format("200601021504.05"), path).Run()

	if utils.IsError(err) {
		return err
	}

	err = exec.Command("touch", "-mt", modificationDate.Format("200601021504.05"), path).Run()

	return err
}

func FileCopy(src, dest string, keepAttributes bool) error {
	if keepAttributes == true && utils.IsUnix { // windows does not support cp nor preserving attributes
		err := exec.Command("cp", "-pRP", src, dest).Run()

		return err
	}
	s, err := os.Open(src)
	if utils.IsError(err) {
		return err
	}

	defer s.Close()
	d, err := os.Create(dest)
	if utils.IsError(err) {
		return err
	}
	if _, err := io.Copy(d, s); utils.IsError(err) {
		d.Close()
		return err
	}
	return d.Close()
}

func FileMove(src, dest string) error {
	err := os.Rename(src, dest)

	if utils.IsError(err) {
		return err
	}

	return nil
}

func MkDirP(dir string) {
	if !PathExists(dir) {
		utils.Catch(os.MkdirAll(dir, DirPerms))
	}
}
