package main

import (
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hirochachacha/go-smb2"
)

type SmbBackupFolder struct {
	rootPerm     fs.FileMode
	rootUrl      *url.URL
	szShare      string
	szRootFolder string

	smbConn    net.Conn
	smbDialer  *smb2.Dialer
	smbSession *smb2.Session
	smbShare   *smb2.Share

	oFile *smb2.File
}

func (bkps *SmbBackupFolder) getUrl() *url.URL {
	return bkps.rootUrl
}

func (bkps *SmbBackupFolder) getPerm() fs.FileMode {
	return bkps.rootPerm
}
func (bkps *SmbBackupFolder) setRootMode(fm fs.FileMode) {
	bkps.rootPerm = fm
}

func (bkps *SmbBackupFolder) getRootFolder() string {
	return bkps.szRootFolder
}

func (bkps *SmbBackupFolder) Stat(name string) (fs.FileInfo, error) {
	return bkps.smbShare.Stat(name)
}
func (bkps *SmbBackupFolder) MkdirAll(path string, perm fs.FileMode) error {
	return bkps.smbShare.MkdirAll(path, perm)
}

func (bkps *SmbBackupFolder) OpenFile(path string, name string) error {
	var err error
	bkps.oFile, err = bkps.smbShare.Open(prepareTargetName(bkps, path, name))
	return err
}
func (bkps *SmbBackupFolder) CreateFile(path string, name string) error {
	var err error
	bkps.oFile, err = bkps.smbShare.Create(prepareTargetName(bkps, path, name))
	return err
}

func (bkps *SmbBackupFolder) ReadFile(buf []byte) (int, error) {
	return bkps.oFile.Read(buf)
}

func (bkps *SmbBackupFolder) WriteFile(buf []byte) (int, error) {
	return bkps.oFile.Write(buf)
}

func (bkps *SmbBackupFolder) CloseFile() error {
	if bkps.oFile != nil {
		err := bkps.oFile.Close()
		bkps.oFile = nil
		return err
	}
	return nil
}

func (bkps *SmbBackupFolder) Close() {
	bkps.smbShare.Umount()
	bkps.smbSession.Logoff()
	bkps.smbConn.Close()
}

func (bkps *SmbBackupFolder) SetParams(path string, name string, modTime time.Time, perm fs.FileMode) error {
	err := bkps.smbShare.Chtimes(prepareTargetName(bkps, path, name), modTime, modTime)
	err2 := bkps.smbShare.Chmod(prepareTargetName(bkps, path, name), perm)
	if err != nil { //we will try and do both but return either error.
		return err
	}
	return err2
}

func (bkps *SmbBackupFolder) DeleteFile(path string, name string) error {
	return bkps.smbShare.Remove(prepareTargetName(bkps, path, name))
}

func (bkps *SmbBackupFolder) RemoveAll(path string) error {
	return bkps.smbShare.RemoveAll(prepareTargetName(bkps, path, ""))
}

func (bkps *SmbBackupFolder) ReadFolder(path string) ([]os.FileInfo, error) {
	fpr, err := bkps.smbShare.Open(path)

	if err != nil {
		return nil, err
	}
	defer fpr.Close()
	return fpr.Readdir(0)
}

func InitializeToPathSmb(szUrl *url.URL, pSrc BackupFolder) BackupFolder {
	var bkps SmbBackupFolder

	bkps.rootUrl = szUrl

	//remove any leading slashes and then cut the string at first '/' to separate the share name from folder path
	str1, str2, _ := strings.Cut(strings.TrimLeft(szUrl.Path, "/"), "/")
	bkps.szShare = str1
	bkps.szRootFolder = str2 //szRootFolder may be empty if the root of the share is supposed to be used for backup

	szPort := szUrl.Port()
	if len(szPort) == 0 {
		szPort = "445"
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", szUrl.Hostname(), szPort))
	if err != nil {
		log.Fatalln("Error connecting to SMB server ", szUrl.Host, " : ", err)
	}
	bkps.smbConn = conn

	username := szUrl.User.Username()
	pass, _ := szUrl.User.Password()

	if len(username) == 0 {
		username = "Guest" //or should it be empty?
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     username,
			Password: pass,
		},
	}

	bkps.smbDialer = d

	s, err := d.Dial(conn)
	if err != nil {
		log.Fatalln("Error dialing in to SMB server ", szUrl.Host, " : ", err)
	}

	bkps.smbSession = s

	fs, err := s.Mount(bkps.szShare)
	if err != nil {
		log.Fatalln("Accessing share ", bkps.szShare, " in SMB server ", szUrl.Host, " : ", err)
	}

	bkps.smbShare = fs

	return &bkps
}
