package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"time"
)

type ZtLog struct {
	logFile *os.File
	openErr error
}

func (ztl *ZtLog) OpenLogFile() {
	hdir, err := os.UserHomeDir()
	if err != nil {
		uname, err := user.Current()
		if err != nil {
			log.Fatalf("Unable to get current username")
		}
		hdir = fmt.Sprintf("/home/%s", uname)
	}
	logPath := fmt.Sprintf("%s%c%s", hdir, os.PathSeparator, ".ztbackup")
	fName := fmt.Sprintf("%04d%02d.log", time.Now().Year(), time.Now().Month())
	fPath := fmt.Sprintf("%s%c%s", logPath, os.PathSeparator, fName)
	os.MkdirAll(logPath, 0755)
	//if it fails, openErr will be non-nil
	ztl.logFile, ztl.openErr = os.OpenFile(fPath, os.O_APPEND|os.O_WRONLY, 0644)
	if ztl.openErr != nil {
		ztl.logFile, ztl.openErr = os.Create(fPath)
	}
	if ztl.openErr != nil {
		fmt.Println("Error opening log ", fPath, " : ", err)
	}
}

func (ztl *ZtLog) Printf(format string, a ...any) {

	outs := fmt.Sprintf(format, a...)

	fmt.Print(outs)
	if ztl.logFile == nil {
		ztl.OpenLogFile()
	}
	if ztl.logFile == nil {
		return
	}
	fmt.Fprint(ztl.logFile, outs)
}

func (ztl *ZtLog) CloseLogFile() {
	if ztl.logFile != nil {
		ztl.logFile.Close()
	}
}
