package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Initialize(szPath string, pSrc BackupFolder) BackupFolder {

	if filepath.IsAbs(szPath) || filepath.IsLocal(szPath) {
		return InitializeToPathLocal(szPath, pSrc)
	}
	//TODO: In case of windows, handle \\server\share kind of URLs.

	//now the remote URLs
	foldURL, err := url.Parse(szPath)
	if err != nil {
		log.Fatalln("Unable to parse Source URL", err)
	}
	switch foldURL.Scheme {
	case "smb":
		return InitializeToPathSmb(foldURL, pSrc)
	case "sftp", "ssh":
		return InitializeToPathSftp(foldURL, pSrc)
	default:
		log.Fatalf("Unknown or invalid source folder/URL scheme '%s'", foldURL.Scheme)
	}
	return nil
}

type BackupFolder interface {
	getPerm() fs.FileMode
	//getUrl() *url.URL
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
	getScanner() *bufio.Scanner

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
	fst, err := bkps.Stat(bkps.getRootFolder())
	if errors.Is(err, fs.ErrNotExist) {
		if pSrc == nil {
			log.Fatalf("Specified local folder '%s' does not exist. Aborting...", bkps.getRootFolder())
		} else {
			log.Printf("Specified destination folder '%s' does not exist. Creating.", bkps.getRootFolder())
			errDir := bkps.MkdirAll(bkps.getRootFolder(), pSrc.getPerm())
			if errDir != nil {
				log.Fatalln("Error creating destination folder: ", errDir)
			}
			fsn, err := bkps.Stat(bkps.getRootFolder()) //stat again. just to make sure.
			if errors.Is(err, fs.ErrNotExist) {
				log.Fatalln("Error creating destination folder: ", err)
			}
			bkps.setRootMode(fsn.Mode())
		}
	} else if err != nil {
		log.Fatalln("Error checking ", getBackupFolderType(pSrc), " folder: ", err)
	} else if fst.IsDir() == false { //exists, but not a folder
		log.Fatalf("'%s' exists but is not a folder.", bkps.getRootFolder())
	} else {
		bkps.setRootMode(fst.Mode())
	}

}

func loadExcludeList(bkps BackupFolder, path string) []string {
	err := bkps.OpenFile(path, ".ztexclude")
	if err != nil {
		return nil
	}
	defer bkps.CloseFile()

	scans := bkps.getScanner()

	//exList := make([]string, 5)
	var exList []string

	scans.Split(bufio.ScanLines)

	for scans.Scan() {
		exItem := strings.TrimSpace(scans.Text())
		if len(exItem) != 0 {
			exList = append(exList, exItem)
		}
	}

	if len(exList) != 0 {
		return exList
	}
	return nil
}
