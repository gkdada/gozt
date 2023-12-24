//go:build darwin
// +build darwin

package main

//serves a list of files to exclude from backup for the specified OS
// these files will be removed from backup if already present.
//
//.~lock files are created by libreoffice when editing. These are machine specific and should not be backed up.

func getOsSpecificExcludes() []string {
	return []string{".localized", 
			"._DS_Store",
			".DS_Store",
			".~lock.*",
		       "Photos Library.photoslibrary",
		       "Music Library.musiclibrary",
			"TV Library.tvlibrary",
		       "Media.localized",
		       ".localized"
		       }
}
