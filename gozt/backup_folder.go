package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"time"
)

func Initialize(szUrl string, pSrc BackupFolder) BackupFolder {
	foldURL, err := url.Parse(szUrl)
	if err != nil {
		log.Fatalf("Unable to parse Source URL", err)
	}
	switch foldURL.Scheme {
	case "smb":
		return InitializeToPathSmb(foldURL, pSrc)
	case "sftp", "ssh":
		return InitializeToPathSftp(foldURL, pSrc)
	case "":
		return InitializeToPathLocal(foldURL, pSrc)
	default:
		log.Fatalf("Unknown or invalid source folder/URL %s", szUrl)
	}
	return nil
}

type BackupFolder interface {
	getPerm() fs.FileMode
	getUrl() *url.URL
	getRootFolder() string

	Stat(name string) (fs.FileInfo, error)
	MkdirAll(path string, perm fs.FileMode) error
	ReadFolder(dirname string) ([]os.FileInfo, error)
	OpenFile(path string, name string) error
	CreateFile(path string, name string) error
	ReadFile(buf []byte) (int, error)
	WriteFile(buf []byte) (int, error)
	CloseFile() error
	DeleteFile(path string, name string) error
	RemoveAll(path string) error
	SetParams(path string, name string, modTime time.Time, perm fs.FileMode) error

	setRootMode(fm fs.FileMode)
	Close()
}

func getBackupFolderType(pSrc BackupFolder) string {
	if pSrc == nil {
		return "Source"
	}
	return "Destination"
}

func prepareTargetName(bkps BackupFolder, path string, name string) string {
	if len(path) == 0 {
		if len(name) == 0 {
			return bkps.getRootFolder()
		}
		return fmt.Sprintf("%s%c%s", bkps.getRootFolder(), os.PathSeparator, name)
	}
	if len(name) == 0 {
		return fmt.Sprintf("%s%c%s", bkps.getRootFolder(), os.PathSeparator, path)
	}
	return fmt.Sprintf("%s%c%s%c%s", bkps.getRootFolder(), os.PathSeparator, path, os.PathSeparator, name)
}

func getFileInfo(bkps BackupFolder, path string, name string) (fs.FileInfo, error) {
	return bkps.Stat(prepareTargetName(bkps, path, name))
}

func ReadDir(bkps BackupFolder, dirname string) ([]os.FileInfo, error) {
	var readex string

	if len(dirname) == 0 {
		readex = bkps.getRootFolder()
	} else {
		readex = fmt.Sprintf("%s%c%s", bkps.getRootFolder(), os.PathSeparator, dirname)
	}
	return bkps.ReadFolder(readex)
}

func checkExists(bkps BackupFolder, pSrc BackupFolder) {
	//check if folder exists.
	fst, err := bkps.Stat(bkps.getUrl().Path)
	if errors.Is(err, fs.ErrNotExist) {
		if pSrc == nil {
			log.Fatalf("Specified local folder '%s' does not exist. Aborting...", bkps.getUrl().Path)
		} else {
			log.Printf("Specified destination folder '%s' does not exist. Creating.", bkps.getUrl().Path)
			errDir := bkps.MkdirAll(bkps.getUrl().Path, pSrc.getPerm())
			if errDir != nil {
				log.Fatalln("Error creating destination folder: ", errDir)
			}
			fsn, err := bkps.Stat(bkps.getUrl().Path) //stat again. just to make sure.
			if errors.Is(err, fs.ErrNotExist) {
				log.Fatalln("Error creating destination folder: ", err)
			}
			bkps.setRootMode(fsn.Mode())
		}
	} else if err != nil {
		log.Fatalln("Error checking ", getBackupFolderType(pSrc), " folder: ", err)
	} else if fst.IsDir() == false { //exists, but not a folder
		log.Fatalf("'%s' exists but is not a folder.", bkps.getUrl().String())
	} else {
		bkps.setRootMode(fst.Mode())
	}

}
