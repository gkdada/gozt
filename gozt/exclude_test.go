package main

import (
	"fmt"
	"testing"
)

func TestExclusions(t *testing.T) {

	bkps := Initialize("/home/gkdada/Downloads/ToSend", nil)

	var zte ztExclude

	zte.LoadFile(bkps, "")

	osSpec := getOsSpecificExcludes()

	fmt.Println("OS Specific = ", len(osSpec))

	for _, rn := range zte.exLocalList {
		fmt.Println("File/Folder: ", rn)
	}

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
