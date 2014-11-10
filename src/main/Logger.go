package main

import (
	"fmt"
	"time"
)

type Logger struct {

}

func getDate() string {
	return time.Now().Format("02-01-2006 15:04:05")
}

func (l* Logger) Debug(str string, args ...interface{}) {
	fmt.Printf("[" + getDate() + "] [DEBUG] " + str + "\n", args...)
}

func (l* Logger) Error(str string, args ...interface{}) {
	fmt.Printf("[" + getDate() + "] [ERROR] " + str + "\n", args...)
}
