package main

import (
	"os"
	"path/filepath"
)

var (
	Reset = "\033[0m"
	Red   = "\033[31m"
	Green = "\033[32m"
	Cyan  = "\033[36m"
	// Yellow = "\033[33m"
	// Blue   = "\033[34m"
	// Purple = "\033[35m"
	// Gray   = "\033[37m"
	// White  = "\033[97m"
)

var (
	executableApp  string
	executablePath string
)

var config Config
var gamesList map[string]Games

func init() {
	var err error
	executableApp, err = os.Executable()
	if err != nil {
		panic(err)
	}
	executablePath = filepath.Dir(executableApp)

	config.ReadConfig()
	gamesList = config.GetList()
}
