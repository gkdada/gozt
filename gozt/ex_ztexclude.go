package main

import (
	"github.com/tzvetkoff-go/fnmatch"
)

type ztExclude struct {
	exLocalList []string
}

func (zte *ztExclude) LoadFile(bkps BackupFolder, path string) error {
	zte.exLocalList = loadExcludeList(bkps, path)

	return nil
}

// If true, do not backup this file/folder. Do not delete already backed up file/folder
func (zte *ztExclude) IsExcluded(fname string) bool {
	if zte.exLocalList == nil { //no .ztexclude found in this folder.
		return zte.IsOsSpecific(fname)
	}

	for _, exThis := range zte.exLocalList {
		if fnmatch.Match(exThis, fname, 0) {
			return true
		}
	}

	return zte.IsOsSpecific(fname)
}

// If true, do not backup this file/folder. Delete if already backed up
func (zte *ztExclude) IsOsSpecific(fname string) bool {
	exList := getOsSpecificExcludes()

	for _, exThis := range exList {
		if fnmatch.Match(exThis, fname, 0) {
			return true
		}
	}

	return false
}
