package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/eiannone/keyboard"
	"golang.org/x/text/message"
)

type BackupOption uint16

const (
	optAsk BackupOption = iota
	optLeave
	optDelete
)

const StdQueryDelaySeconds = 120         //seconds
const MinHalvingDelay = time.Duration(4) //minimum delay of 4 seconds. If the QueryDelay is more than this, we halve it.

type ztStatistics struct {
	NumFolders int64

	NumFilesSkipped  int64
	SizeFilesSkipped int64

	NumFilesCopied  int64
	SizeFilesCopied int64

	NumFilesDeleted int64
	//SizeFilesDeleted int64

	NumFilesRestored  int64
	SizeFilesRestored int64
}

type Backup struct {
	FileOption        BackupOption
	FolderOption      BackupOption
	RecursiveFlag     bool
	QueryDelaySeconds time.Duration //starts with 120 seconds, halves with every timeout until
	Statistics        ztStatistics
	ztl               ZtLog

	folderSkipCount  int
	statPrinter      *message.Printer
	srcBack, dstBack *BackupFolder
}

func (bkp *Backup) ProcessFlags(flags string) {
	i := 0
	if flags[i] == '-' {
		i++
	}
	for _, ctr := range flags {
		switch ctr {
		case 'a':
			bkp.FileOption = optAsk
		case 'b':
			bkp.FolderOption = optAsk
		case 'l':
			bkp.FileOption = optLeave
		case 'm':
			bkp.FolderOption = optLeave
		case 'd':
			bkp.FileOption = optDelete
		case 'e':
			bkp.FolderOption = optDelete
		case 'r':
			bkp.RecursiveFlag = true

		}
	}
}

var copyBuffer []byte

// we try 10Meg buffer size
const COPY_BUFFERSIZE = 512000

func (bkp *Backup) LogPrintf(format string, a ...any) {

	bkp.ztl.Printf(format, a...)
}

func (bkp *Backup) StartBackup(src *BackupFolder, dst *BackupFolder) error {

	bkp.QueryDelaySeconds = StdQueryDelaySeconds

	bkp.srcBack = src
	bkp.dstBack = dst

	copyBuffer = make([]byte, COPY_BUFFERSIZE)

	return bkp.recurseBackup("")
}

const progress_wheel = "|/-\\"

func (bkp *Backup) recurseBackup(folderPath string) error {

	if len(folderPath) != 0 {
		fmt.Printf("\rProcessing folder %s\r\n", folderPath)
	} else {
		bkp.statPrinter = message.NewPrinter(message.MatchLanguage("en")) //for now, we default to English (since all our messages are in English anyway)
		bkp.LogPrintf("\rStarted at %s\r\n", time.Now().Format(time.UnixDate))
		defer bkp.printStatistics()
		//"Ended at" now moved to printStatistics
		//defer fmt.Println("\rEnded at ", time.Now().Format(time.UnixDate))
	}
	fmts, err := ReadDir(*bkp.srcBack, folderPath)

	if err != nil {
		return err
	}

	bkp.Statistics.NumFolders++

	bkp.folderSkipCount = 0
	srcInfo, err := getFileInfo(*bkp.srcBack, folderPath, "")
	err = bkp.ensurePath(*bkp.dstBack, folderPath, srcInfo.Mode())
	if err != nil {
		fmt.Println("\rError creating path for ", folderPath, err)
		return err
	}

	//1. for each file in source, backup as required.
	for _, ctr := range fmts {
		//log.Printf("ctr: %s \t\t%s", ModeString(ctr), ctr.Name())
		if ctr.Mode().IsRegular() {
			bkp.processRegularFile(ctr, folderPath, true)
		}
	}

	//2. for each file in destination, check source
	fmtd, err := ReadDir(*bkp.dstBack, folderPath)

	if err != nil {
		return err
	}

	for _, ctr := range fmtd {
		//log.Printf("ctr: %s \t\t%s", ModeString(ctr), ctr.Name())
		if ctr.IsDir() && bkp.RecursiveFlag {
			_, err := getFileInfo(*bkp.srcBack, folderPath, ctr.Name())
			if errors.Is(err, fs.ErrNotExist) {
				status := bkp.fileMissingQuestion(folderPath, ctr)
				switch status {
				case copyDeleteDestination:
					bkp.recurseDelete(*bkp.dstBack, bkp.prepareName(folderPath, ctr.Name()))
				case copyBackward:
					bkp.recurseRestore(bkp.prepareName(folderPath, ctr.Name()), ctr)
				case copyLeave:
				}
			}

		} else if ctr.Mode().IsRegular() {
			bkp.processRegularFile(ctr, folderPath, false)
		}
	}

	//3. for each folder in source, recurse
	for _, ctr := range fmts {
		//log.Printf("ctr: %s \t\t%s", ModeString(ctr), ctr.Name())
		if ctr.IsDir() && bkp.RecursiveFlag {
			/*err :=*/
			bkp.recurseBackup(bkp.prepareName(folderPath, ctr.Name()))
		}
	}

	return nil
}

func (bkp *Backup) recurseRestore(folderPath string, srcInfo fs.FileInfo) {
	//since the folder doesn't exist, we just restore everything full speed
	err := bkp.ensurePath(*bkp.srcBack, folderPath, srcInfo.Mode())
	fmtd, err := ReadDir(*bkp.dstBack, folderPath)

	if err != nil {
		return
	}
	//for each file
	for _, ctr := range fmtd {
		if ctr.Mode().IsRegular() {
			bkp.copyFile(folderPath, ctr, false)
		}
	}
	//for each folder
	for _, ctr := range fmtd {
		//log.Printf("ctr: %s \t\t%s", ModeString(ctr), ctr.Name())
		if ctr.IsDir() {
			bkp.recurseRestore(bkp.prepareName(folderPath, ctr.Name()), ctr)
		}
	}

}

func (bkp *Backup) recurseDelete(bkps BackupFolder, folderName string) {
	bkps.RemoveAll(folderName)
}

func (bkp *Backup) prepareName(path string, name string) string {
	if len(path) == 0 {
		return name
	}
	return fmt.Sprintf("%s%c%s", path, os.PathSeparator, name)
}

type copyType uint16

const (
	copyForward           copyType = iota //source -> destination
	copyBackward                          //destination -> source
	copyLeave                             //leave the destination file /source file alone
	copyDeleteDestination                 //delete the destination
)

func (bkp *Backup) processRegularFile(fStart fs.FileInfo, path string, bForward bool) error {
	status := copyLeave
	//1. Does the file exist in destination?
	if bForward {
		fDst, err := getFileInfo(*bkp.dstBack, path, fStart.Name())
		if errors.Is(err, fs.ErrNotExist) {
			//fmt.Printf("File %s does not exist.\r\n", bkp.prepareName(path, fStart.Name()))
			status = copyForward
		} else {
			status = bkp.copyCheck(path, fStart, fDst)
		}
	} else {
		_, err := getFileInfo(*bkp.srcBack, path, fStart.Name())
		if errors.Is(err, fs.ErrNotExist) {
			status = bkp.fileMissingQuestion(path, fStart)
		}
		//if the file exists, no action during backward check
	}

	switch status {
	case copyForward:
		return bkp.copyFile(path, fStart, true)
	case copyBackward:
		return bkp.copyFile(path, fStart, false)
	//case copyDeleteSource:
	//	bkp.Statistics.NumFilesDeleted++
	//	return (*bkp.srcBack).DeleteFile(path, fStart.Name())
	case copyDeleteDestination:
		bkp.Statistics.NumFilesDeleted++
		return (*bkp.dstBack).DeleteFile(path, fStart.Name())
	default:
		fmt.Printf("\rSkipping...%c", progress_wheel[bkp.folderSkipCount%4])
		bkp.folderSkipCount++
		if bForward {
			bkp.Statistics.NumFilesSkipped++
			bkp.Statistics.SizeFilesSkipped += fStart.Size()
		}
	}
	return nil
}

func (bkp *Backup) ensurePath(bkps BackupFolder, path string, perm fs.FileMode) error {
	if len(path) == 0 { //we've already ensured this path exists before.
		return nil
	}
	//fmt.Printf("Ensuring path %s\r\n", path)
	return bkps.MkdirAll(prepareTargetName(bkps, path, ""), perm)
}

func (bkp *Backup) copyCheck(path string, fSrc fs.FileInfo, fDst fs.FileInfo) copyType {
	//TODO: add exclusion (.ztexclude) check

	//fmt.Printf("copyCheck for %s.\r\n", bkp.prepareName(path, fSrc.Name()))

	if fSrc.ModTime().After(fDst.ModTime()) {
		//fmt.Println("Destination time: ", fDst.ModTime())
		//fmt.Println("     Source time: ", fSrc.ModTime())
		Diff := fSrc.ModTime().Sub(fDst.ModTime())
		//enough to ignore smb/ssh timestamp copy errors etc.
		if Diff > (time.Second * 6) {
			return copyForward
		}
	}

	if fSrc.ModTime().Before(fDst.ModTime()) {
		Diff := fDst.ModTime().Sub(fSrc.ModTime())
		//enough to fix smb copy errors etc.
		if Diff > (time.Second * 6) {
			return bkp.fileRestoreQuestion(path, fSrc, fDst)
		}
	}

	//now, the mod time is same (or about the same). is the size different? Then we will go ahead and copy.
	if fSrc.Size() != fDst.Size() {
		return copyForward
	}
	return copyLeave
}

// This is used if a (backed up) file is missing in source.
func (bkp *Backup) fileMissingQuestion(path string, fDst fs.FileInfo) copyType {

	szItemType := "file"
	if fDst.IsDir() {
		szItemType = "folder"
		if bkp.FolderOption == optLeave {
			return copyLeave
		} else if bkp.FolderOption == optDelete {
			return copyDeleteDestination
		}
	} else {
		if bkp.FileOption == optLeave {
			return copyLeave
		} else if bkp.FileOption == optDelete {
			return copyDeleteDestination
		}
	}

	fmt.Printf("\rThe source for the backed up %s '%s' doesn't exist anymore.\r\n", szItemType, bkp.prepareName(path, fDst.Name()))
	szQueryString := fmt.Sprintf("Do you want to (d)elete, (r)estore, or (l)eave the %s or [q]uit?", szItemType)
	ans := bkp.OneCharAnswer(szQueryString, "drl", 'l')
	switch ans {
	case 'd':
		return copyDeleteDestination
	case 'r':
		return copyBackward
	default:
		return copyLeave
	}
}

func (bkp *Backup) fileRestoreQuestion(path string, fSrc fs.FileInfo, fDst fs.FileInfo) copyType {

	if bkp.FileOption == optLeave {
		return copyLeave
	}

	bkp.statPrinter.Println("\rThe destination for the backed up file '", bkp.prepareName(path, fSrc.Name()), "' is newer than the source.\r\n\r\n                size (bytes)            modified time\r\n")
	bkp.statPrinter.Printf("source:      %26d %s\r\n", fSrc.Size(), fSrc.ModTime().String())
	bkp.statPrinter.Printf("destination: %26d %s\r\n\r\n", fDst.Size(), fDst.ModTime().String())
	//fmt.Printf("Do you want to (b)ackup, (r)estore, or (l)eave the file?")

	ans := bkp.OneCharAnswer("Do you want to (b)ackup, (r)estore, or (l)eave the file or [q]uit?", "brl", 'l')
	switch ans {
	case 'b':
		return copyForward
	case 'r':
		return copyBackward
	default:
		return copyLeave
	}
}

func (bkp *Backup) OneCharAnswer(Query string, Answers string, defaultAnswer rune) rune {

	defer keyboard.Close()

	keyin, err := keyboard.GetKeys(10)
	if err != nil {
		fmt.Printf("Error getting key events.\r\n")
		return defaultAnswer
	}

	fmt.Println(Query)

	do_until := time.Now().Add(time.Duration(bkp.QueryDelaySeconds * time.Second))

	for {
		select {
		//char, _, err := keyboard.GetSingleKey()
		case event := <-keyin:
			if event.Err != nil {
				fmt.Printf("Error getting keyboard input. Taking default action.\r\n")
				return defaultAnswer
			}
			char := event.Rune
			char = unicode.ToLower(char)
			if char == 'q' {
				keyboard.Close()
				os.Exit(0)
			}
			if strings.Contains(Answers, string(char)) {
				return char
			}
			break
		default:
			if time.Now().Before(do_until) {
				fmt.Printf("\r[%c] in %d seconds: ", defaultAnswer, int(do_until.Sub(time.Now()).Seconds()))
				time.Sleep(time.Millisecond * 20)
			} else {
				//timeout occurred
				if bkp.QueryDelaySeconds > MinHalvingDelay {
					bkp.QueryDelaySeconds = bkp.QueryDelaySeconds / 2
				}
				return defaultAnswer
			}
		}
	}
	return defaultAnswer
}

// func (bkp *Backup) OneCharAnswer(Query string, Answers string, defaultAnswer rune) rune {
// 	bExit := false

// 	var pushx = keyboard.KeyEvent{
// 		Key:  0,
// 		Rune: defaultAnswer,
// 		Err:  nil,
// 	}

// 	go func() {
// 		do_until := time.Now().Add(time.Duration(bkp.QueryDelaySeconds * time.Second))
// 		for time.Now().Before(do_until) {
// 			fmt.Printf("\r%s: [%c] in %d seconds ", Query, defaultAnswer, int(do_until.Sub(time.Now()).Seconds()))
// 			time.Sleep(time.Millisecond * 20)
// 			if bExit {
// 				return
// 			}
// 		}
// 		//timeout occurred.
// 		if bkp.QueryDelaySeconds > MinHalvingDelay {
// 			bkp.QueryDelaySeconds = bkp.QueryDelaySeconds / 2
// 		}
// 		keyboard.PushKey(pushx)
// 		return
// 	}()

// 	for {
// 		char, _, err := keyboard.GetSingleKey()
// 		if err == nil {
// 			char = unicode.ToLower(char)
// 			if char == 'q' {
// 				fmt.Printf("\r\nExiting...")
// 				os.Exit(0)
// 			}
// 			if strings.Contains(Answers, string(char)) {
// 				bExit = true
// 				return char
// 			}
// 		}
// 	}
// 	return defaultAnswer
// }

func (bkp *Backup) copyFile(path string, fi fs.FileInfo, bForward bool) error {

	var strAction string
	var bkFrom, bkTo BackupFolder

	if bForward {
		bkFrom = *bkp.srcBack
		bkTo = *bkp.dstBack
		strAction = fmt.Sprintf("\rCopying %s...", bkp.prepareName(path, fi.Name()))
	} else {
		bkFrom = *bkp.dstBack
		bkTo = *bkp.srcBack
		strAction = fmt.Sprintf("\rRestoring %s...", bkp.prepareName(path, fi.Name()))
	}
	//fss, _ := getFileInfo(bkFrom, path, "")
	//err := bkp.ensurePath(bkTo, path, fss.Mode())
	//if err != nil {
	//	log.Println("Error ensuring path", path, " exists in destination : ", err)
	//	return err
	//}

	err := bkFrom.OpenFile(path, fi.Name())
	if err != nil {
		fmt.Println("\rError opening source file ", bkp.prepareName(path, fi.Name()), " : ", err)
		return err
	}

	defer bkFrom.CloseFile()

	err = bkTo.CreateFile(path, fi.Name())
	if err != nil {
		fmt.Println("\rError creating/opening destination file ", bkp.prepareName(path, fi.Name()), " : ", err)
		return err
	}
	err = bkp.copyFileContents(bkp.prepareName(path, fi.Name()), bkFrom, bkTo, strAction, fi.Size())
	//if copy fails delete the file so that it won't be left with half-finished job.
	bkTo.CloseFile()
	if err != nil {
		bkTo.DeleteFile(path, fi.Name())
		return err
	}

	if bForward {
		bkp.Statistics.NumFilesCopied++
		bkp.Statistics.SizeFilesCopied += fi.Size()
	} else {
		bkp.Statistics.NumFilesRestored++
		bkp.Statistics.SizeFilesRestored += fi.Size()
	}
	//set mode and time
	return bkTo.SetParams(path, fi.Name(), fi.ModTime(), fi.Mode())
}

func (bkp *Backup) copyFileContents(filePath string, bkFrom BackupFolder, bkTo BackupFolder, strAction string, sizeEstimate int64) error {

	var nTotal int

	for {
		n, err := bkFrom.ReadFile(copyBuffer)
		if err != nil && err != io.EOF {
			fmt.Println("\rRead error copying ", filePath, " : ", err)
			return err
		}
		if n == 0 {
			break
		}
		if _, err := bkTo.WriteFile(copyBuffer[:n]); err != nil {
			fmt.Println("\rWrite error copying ", filePath, " : ", err)
			return err
		}
		nTotal += n
		fmt.Printf("%s%d%%", strAction, (nTotal*100)/int(sizeEstimate))
		//fmt.Printf("Copied %d bytes\r\n", nTotal)
	}

	fmt.Printf("%sdone\r\n", strAction)

	return nil
}

func (bkp *Backup) printStatistics() {
	//char PrintBuf[4000];

	bkp.LogPrintf("\r              \r\nEnded at %s\r\n", time.Now().Format(time.UnixDate))

	statful := bkp.statPrinter.Sprintf("\r\nFolders traversed            %15d\r\n", bkp.Statistics.NumFolders)
	statful += bkp.statPrinter.Sprintf("Files skipped                %15d\r\n", bkp.Statistics.NumFilesSkipped)
	if bkp.Statistics.SizeFilesSkipped != 0 {
		statful += bkp.statPrinter.Sprintf("Size of files skipped        %15d octets\r\n", bkp.Statistics.SizeFilesSkipped)
	}
	statful += bkp.statPrinter.Sprintf("Files copied                 %15d\r\n", bkp.Statistics.NumFilesCopied)
	if bkp.Statistics.SizeFilesCopied != 0 {
		statful += bkp.statPrinter.Sprintf("Size of files copied         %15d octets\r\n", bkp.Statistics.SizeFilesCopied)
	}
	statful += bkp.statPrinter.Sprintf("Files restored               %15d\r\n", bkp.Statistics.NumFilesRestored)
	if bkp.Statistics.SizeFilesRestored != 0 {
		statful += bkp.statPrinter.Sprintf("Size of files restored       %15d octets\r\n", bkp.Statistics.SizeFilesRestored)
	}
	statful += bkp.statPrinter.Sprintf("Files deleted                %15d\r\n\r\n", bkp.Statistics.NumFilesDeleted)

	bkp.LogPrintf(statful)
}
