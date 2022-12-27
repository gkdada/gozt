package main

import (
	"fmt"
	"os"
	"time"
)

func Fatalln(line string) {
	fmt.Println(line)
	os.Exit(1)
}

func main() {
	//backups.Ssh_init()
	//backups.Smb_init()

	var Src, Dst string

	var bkp Backup

	bkp.LogPrintf("ztbackup on Go. (c) gkdada 2022-2022\r\n")

	for i, ctr := range os.Args {
		if i == 0 {
			//skip. this is the program name
		} else if ctr[0] == '-' {
			bkp.ProcessFlags(ctr)
		} else if len(Src) == 0 {
			Src = ctr
		} else if len(Dst) == 0 {
			Dst = ctr
		} else {
			bkp.LogPrintf("\r\nToo many string parameters (%s). Expecting only source, destination and flags", ctr)
			os.Exit(1)
		}
	}

	if len(Src) == 0 {
		bkp.LogPrintf("\r\nMissing source folder/URL")
		os.Exit(1)
	} else if len(Dst) == 0 {
		bkp.LogPrintf("\r\nMissing destination folder/URL")
		os.Exit(1)
	}

	bkp.LogPrintf("Initiating zero-touch backup at %s\r\n", time.Now().Format(time.UnixDate))
	bkp.LogPrintf("Source Folder: %s\r\n", Src)
	bkp.LogPrintf("Destination Folder: %s\r\n", Dst)

	srcBack := Initialize(Src, nil)

	dstBack := Initialize(Dst, srcBack)

	defer srcBack.Close()
	defer dstBack.Close()

	bkp.StartBackup(&srcBack, &dstBack)

}
