package main

import (
	"bufio"
	"io/fs"
	"log"
	"os"
	"time"
)

type LocalBackupFolder struct {
	rootPerm fs.FileMode
	//rootUrl  *url.URL

	szRootPath string

	oFile *os.File
}

func (bkps *LocalBackupFolder) getPerm() fs.FileMode {
	return bkps.rootPerm
}

func (bkps *LocalBackupFolder) getRootFolder() string {
	return bkps.szRootPath
	//return bkps.rootUrl.Path
}

func (bkps *LocalBackupFolder) setRootMode(fm fs.FileMode) {
	bkps.rootPerm = fm
}

func (bkps *LocalBackupFolder) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}
func (bkps *LocalBackupFolder) MkdirAll(path string, perm fs.FileMode) error {

	return os.MkdirAll(path, perm)
}

func (bkps *LocalBackupFolder) ReadFolder(path string) ([]os.FileInfo, error) {
	fpr, err := os.Open(path)

	if err != nil {
		return nil, err
	}
	defer fpr.Close()
	return fpr.Readdir(0)
}

func (bkps *LocalBackupFolder) OpenFile(path string, name string) error {
	var err error
	bkps.oFile, err = os.Open(prepareTargetName(bkps, path, name))
	return err
}
func (bkps *LocalBackupFolder) CreateFile(path string, name string) error {
	var err error
	bkps.oFile, err = os.Create(prepareTargetName(bkps, path, name))
	return err
}

func (bkps *LocalBackupFolder) ReadFile(buf []byte) (int, error) {
	return bkps.oFile.Read(buf)
}

func (bkps *LocalBackupFolder) WriteFile(buf []byte) (int, error) {
	return bkps.oFile.Write(buf)
}

func (bkps *LocalBackupFolder) CloseFile() error {
	if bkps.oFile != nil {
		err := bkps.oFile.Close()
		bkps.oFile = nil
		return err
	}
	return nil
}

func (bkps *LocalBackupFolder) SetParams(path string, name string, modTime time.Time, perm fs.FileMode) error {

	pathstring := prepareTargetName(bkps, path, name)
	err := os.Chtimes(pathstring, modTime, modTime)
	err2 := os.Chmod(pathstring, perm)
	if err != nil { //we will try and do both but return either error.
		return err
	}
	return err2
}

func (bkps *LocalBackupFolder) getScanner() *bufio.Scanner {
	return bufio.NewScanner(bkps.oFile)
}

func InitializeToPathLocal(szPath string, pSrc BackupFolder) BackupFolder {
	//just open the specified folder. if failed, probably no access?
	var bkps LocalBackupFolder

	bkps.szRootPath = szPath

	//bkps.rootUrl = szUrl

	checkExists(&bkps, pSrc)

	//check if folder exists.
	fst, err := os.Stat(szPath)
	if os.IsNotExist(err) {
		if pSrc == nil {
			log.Fatalf("Specified local folder '%s' does not exist. Aborting...", szPath)
		} else {
			log.Printf("Specified destination folder '%s' does not exist. Creating.", szPath)
			errDir := os.MkdirAll(szPath, pSrc.getPerm())
			if errDir != nil {
				log.Fatalln("Error creating destination folder: ", errDir)
			}
			fsn, err := os.Stat(szPath) //stat again. just to make sure.
			if os.IsNotExist(err) {
				log.Fatalln("Error creating destination folder: ", err)
			}
			bkps.rootPerm = fsn.Mode()
		}
	} else if err != nil {
		log.Fatalln("Error checking ", getBackupFolderType(pSrc), " folder: ", err)
	} else if fst.IsDir() == false { //exists, but not a folder
		log.Fatalf("'%s' exists but is not a folder.", szPath)
	} else {
		bkps.rootPerm = fst.Mode()
	}
	//if it doesn't AND pSrc is present, create the folder and add the

	return &bkps
}

func (bkps *LocalBackupFolder) DeleteFile(path string, name string) error {
	return os.Remove(prepareTargetName(bkps, path, name))
}

func (bkps *LocalBackupFolder) RemoveAll(path string) error {
	return os.RemoveAll(prepareTargetName(bkps, path, ""))
}

func (bkps *LocalBackupFolder) Close() {
	//nothing to do here since we didn't open a connection
}
