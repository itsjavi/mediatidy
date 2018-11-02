package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"strconv"
)

func getMapValueByKey(dataMap RawJsonMap, key string) string {
	if val, ok := dataMap[key]; ok {
		return sanitizeType(val)
	}

	return ""
}

func sanitizeType(val interface{}) string {
	switch val.(type) {
	case int:
		return strconv.Itoa(val.(int))
	case float64:
		return strconv.FormatFloat(val.(float64), 'f', 6, 64)
	default:
		return fmt.Sprintf("%s", val)
	}
}

func catch(e error, data ... interface{}) {
	if e != nil {
		fmt.Printf("%s\n", data)
		panic(e)
	}
}

func logln(message string, a ...interface{}) {
	fmt.Printf("["+AppName+"] "+message+"\n", a...)
}

func getFileChecksum(path string) (string, error) {
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

func pathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
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
	if !pathExists(dir) {
		catch(os.MkdirAll(dir, PathPerms))
	}
}
