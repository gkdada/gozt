package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/url"
	"os"
	"os/user"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// implementation of BackupFolder with Ssh
type SftpBackupFolder struct {
	rootPerm fs.FileMode
	rootUrl  *url.URL

	sshClient  *ssh.Client
	sftpClient *sftp.Client

	oFile *sftp.File
}

func (bkps *SftpBackupFolder) getPerm() fs.FileMode {
	return bkps.rootPerm
}

// func (bkps *SftpBackupFolder) getUrl() *url.URL {
// 	return bkps.rootUrl
// }

func (bkps *SftpBackupFolder) getRootFolder() string {
	return bkps.rootUrl.Path
}

func (bkps *SftpBackupFolder) setRootMode(fm fs.FileMode) {
	bkps.rootPerm = fm
}

func (bkps *SftpBackupFolder) Stat(name string) (fs.FileInfo, error) {
	return bkps.sftpClient.Stat(name)
}

func (bkps *SftpBackupFolder) MkdirAll(path string, perm fs.FileMode) error {
	return bkps.sftpClient.MkdirAll(path)
}

func (bkps *SftpBackupFolder) ReadFolder(path string) ([]os.FileInfo, error) {
	return bkps.sftpClient.ReadDir(path)
}

func (bkps *SftpBackupFolder) OpenFile(path string, name string) error {
	var err error
	bkps.oFile, err = bkps.sftpClient.Open(prepareTargetName(bkps, path, name))
	return err
}
func (bkps *SftpBackupFolder) CreateFile(path string, name string) error {
	var err error
	bkps.oFile, err = bkps.sftpClient.Create(prepareTargetName(bkps, path, name))
	return err
}

func (bkps *SftpBackupFolder) ReadFile(buf []byte) (int, error) {
	return bkps.oFile.Read(buf)
}

func (bkps *SftpBackupFolder) WriteFile(buf []byte) (int, error) {
	return bkps.oFile.Write(buf)
}

func (bkps *SftpBackupFolder) CloseFile() error {
	if bkps.oFile != nil {
		err := bkps.oFile.Close()
		bkps.oFile = nil
		return err
	}
	return nil
}

func (bkps *SftpBackupFolder) getScanner() *bufio.Scanner {
	return bufio.NewScanner(bkps.oFile)
}

func (bkps *SftpBackupFolder) Close() {
	bkps.sftpClient.Close()
	bkps.sshClient.Close()
}

func (bkps *SftpBackupFolder) SetParams(path string, name string, modTime time.Time, perm fs.FileMode) error {
	err := bkps.sftpClient.Chtimes(prepareTargetName(bkps, path, name), modTime, modTime)
	err2 := bkps.sftpClient.Chmod(prepareTargetName(bkps, path, name), perm)
	if err != nil { //we will try and do both but return either error.
		return err
	}
	return err2
}

func (bkps *SftpBackupFolder) DeleteFile(path string, name string) error {
	return bkps.sftpClient.Remove(prepareTargetName(bkps, path, name))
}

func (bkps *SftpBackupFolder) RemoveAll(path string) error {
	return bkps.sftpClient.RemoveAll(prepareTargetName(bkps, path, ""))
}

func InitializeToPathSftp(szRoot *url.URL, pSrc BackupFolder) BackupFolder {
	var bkps SftpBackupFolder

	bkps.rootUrl = szRoot

	szPort := szRoot.Port()
	if len(szPort) == 0 {
		szPort = "22"
	}

	//TODO: Sanity check. Make sure 'username' is present, server name is present.

	//0. We have to have a user.
	if len(szRoot.User.Username()) == 0 {
		log.Fatalf("Missing username for SSH/SFTP in %s folder.", getBackupFolderType(pSrc))
	}

	passString, isPassword := szRoot.User.Password()

	var conf *ssh.ClientConfig

	if !isPassword {
		//use RSA keys
		hdir, err := os.UserHomeDir()
		if err != nil {
			uname, err := user.Current()
			if err != nil {
				log.Fatalf("Unable to get current username")
			}
			hdir = fmt.Sprintf("/home/%s", uname)
		}
		fmt.Printf("HomeDirectory: %s\r\n", hdir)
		hdir = fmt.Sprintf("%s/.ssh/id_rsa", hdir)
		key, err := os.ReadFile(hdir)
		if err != nil {
			log.Fatalf("unable to read private SSH/RSA key: %v", err)
		}
		// Create the Signer for this private key.
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			log.Fatalf("unable to parse private key: %v", err)
		}
		conf = &ssh.ClientConfig{
			User: szRoot.User.Username(),
			Auth: []ssh.AuthMethod{
				// Use the PublicKeys method for remote authentication.
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
		}
	} else {
		//use pass authentication
		conf = &ssh.ClientConfig{
			User: szRoot.User.Username(),
			Auth: []ssh.AuthMethod{
				ssh.Password(passString),
			},
			HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
		}
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", szRoot.Hostname(), szPort), conf)
	if err != nil {
		log.Fatalln("Failed to connect to ", szRoot.Host, " : ", err)
	}
	bkps.sshClient = client

	ft_conn, err2 := sftp.NewClient(client)
	if err2 != nil {
		log.Fatal("Failed to connect to SFTP: ", err)
	}

	bkps.sftpClient = ft_conn

	checkExists(&bkps, pSrc)

	return &bkps
}
