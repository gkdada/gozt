package main

import (
	"fmt"
	"os"
	"testing"
)

///update (11/7/23): Fixing the test routine.
// One folder that is common to all major OSes (Linux, MacOS, Windows) is HomeFolder->Downloads.
// So the test routine will use THAT folder to check for excluded files and folders.

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true //if it exists but we cannot access it, it is different.
}

func TestExclusions(t *testing.T) {

	hdir, err := os.UserHomeDir()

	if err != nil {
		fmt.Println("Error locating home folder.", err)
		return
	}

	chkPath := fmt.Sprintf("%s%c%s", hdir, os.PathSeparator, "Downloads")
	if pathExists(chkPath) == false {
		chkPath = hdir
		if pathExists(chkPath) == false {
			fmt.Println("Not continuing with the test as home folder is not accessible. May be running in a pod/with a system account or something")
			return
		}
	}
	fmt.Println("Testing folder ", chkPath)

	bkps := Initialize(chkPath, nil)

	var zte ztExclude

	zte.LoadFile(bkps, "")

	osSpec := getOsSpecificExcludes()

	fmt.Println("\r\nOS Specific = ", len(osSpec))

	for _, rn := range osSpec {
		fmt.Println("  File/Folder: ", rn)
	}
	fmt.Println("")

	fmt.Println("Local excludes (.ztexclude) = ", len(zte.exLocalList))
	for _, rn := range zte.exLocalList {
		fmt.Println("  File/Folder: ", rn)
	}
	fmt.Println("")

	srn, err := ReadDir(bkps, "")

	if err != nil {
		t.Errorf("Error reading folder")
	}
	for _, tst := range srn {
		if zte.IsExcluded(tst.Name()) {
			fmt.Println("Skip        : ", tst.Name())
		} else {
			fmt.Println("Process     : ", tst.Name())
		}
	}
}
