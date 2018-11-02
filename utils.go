package main

import (
	"fmt"
	"strconv"
)

func catch(e error, data ... interface{}) {
	if e != nil {
		fmt.Printf("%s\n", data)
		panic(e)
	}
}

func safeString(val interface{}) string {
	switch val.(type) {
	case int:
		return strconv.Itoa(val.(int))
	case float64:
		return strconv.FormatFloat(val.(float64), 'f', 6, 64)
	default:
		return fmt.Sprintf("%s", val)
	}
}

func logLn(message string, a ...interface{}) {
	fmt.Printf("["+AppName+"] "+message+"\n", a...)
}

func logSameLn(format string, args ... interface{}) {
	fmt.Printf("\033[2K\r"+format, args...)
}
