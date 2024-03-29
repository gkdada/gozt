# gozt

## Introduction


This tool allows you to backup folders from/to local drives, samba shares and over ssh/sftp using sync method (i.e. only the newer files are copied). Optionally recursive.

The source or destination can be a local drive (relative or absolute path), samba share or an ssh address. 

A folder in a samba share must be specified in the form of "smb://username:password@server-IP-or-Name/[Path]".

A folder in an remote ssh location must be specified in the form of "sftp://username[:password]@server_ip_or_name[:port]/path". However, we do not recommend providing plain text passwords in command line or in shell script files. It is much safer to add the client's keys to server's 'authorized_keys' so that the client can login without the need for a password. gozt automatically searches for and use .ssh/id_rsa for sftp communication in the absence of password.



## Command Syntax

    gozt [arguments] source-folder destination-folder

* If either source-folder or destination-folder has spaces, you need to enclose the folder name in double quotes. 
* arguments can be combined in to a single parameter. For example, "-a","-b" and "-r" can be combined to "-abr".

### Arguments:

 -r  Recursively back up the contents of source-folder into destination-folder.

 -a  Ask before deleting backed up files when the source for that file no longer exists. You can also optinally restore the file back to the source location. The question times out after a certain number of seconds and backup process continues leaving the file intact.

 -b  Ask before deleting backed up folders when the source for that folder no longer exists. You can also optionally restore the folder back to the source location. Question times out similar to "-a" option.

 -l  Leave a backed up file alone when the source for that file no longer exists. Do not delete the file.

 -m  Leave a backed up folder alone when the source for that file no longer exists. Do not delete the folder.

 -d  Delete a backed up file automatically when the source for that file no longer exists.

 -e  Delete a backed up folder automatically when the source for that folder no longer exists.

### for future implementation

 -n  Do not follow symbolic links when backing up a file or a folder.
 
 -u  Use anonymous access for any samba share in the source and/or destination folders.

### examples

#### from local to remote

    gozt -abr ~/Documents ssh://myuser@10.2.3.4/MyBackups/Documents
    gozt -abr /home/shared/DevCmd sftp://otheruser@192.168.9.4/CodeBackup/DevCmd
    gozt -abr /home/myuser/Videos smb://smbclient@192.168.2.4/FirstShare/Videos

#### from remote to local
    gozt -abr ssh://shared@192.168.5.6/Shared/ShareDocs ~/Documents/ShareDocs

#### for USB hard drives

It is recommended that you create a shell script on the USB drive with the target folder specified relative to 'current' folder
    gozt -abr ~/Documents ./Backups/Documents

#### for cron jobs
    gozt -lmr /home/myuser/Pictures ssh://myuser@10.2.3.4/MyBackups/Pictures

### notes

* Use of options -d and -e are NOT RECOMMENDED since they will result in losing back-up files for source that may have been accidentally deleted.
* Options -l and -m are meant used in scripts (especially the ones that are run as cron jobs)  when user interactions are not possible. 
* The most frequently used combinations of arguments are "-lmr" and "-abr". options "-lmr" will give you a recursive backup of a folder while leaving the backups for any deleted files or folders intact. "-abr" will give you a recursive backup of a folder while asking whether you want to delete (or restore or leave) the backup for the deleted files or folders.
* The combination "-lmr" is convenient for running in an automatically run script (as in a cron job). The same command can then be run manually with "-abr" option to delete the backup copies of intentionally deleted files and folders.


### Exclude files or folders

gozt will support excluding one or more folders and/or files at any level from backup. Salient points:
*  You can create a file called .ztexclude in any folder at any level starting from source-folder down. This file can contain one or more names of folders or files present in this folder, ONE PER LINE. No need to enclose the file/folder names in quotes, since only one name is allowed per line. All folders and files with this name IN THE CURRENT FOLDER will be excluded from the backup.
*  A .ztexclude file applies ONLY TO THE CURRENT FOLDER. Not to its sub-folders.
*  If a file/folder by that name already exists in destination-folder, they will not be deleted. You will have to do it manually if you want to free up that space.
*  Try not to add too many lines to a single .ztexclude file since it has to be loaded into memory in its entirety for the duration of the processing of that folder (and its sub-folders).
*  All shell wildcards are supported (?, *, [...] and [!...])


### OS Specific excludes

Each OS will have certain files that are machine specific and/or session specific and these should not be backed up or restored.

There is an ex_list_linux.go which returns items like ".~lock.*" (which is a lock file created by libreoffice applications) and "lost+found". 

An ex_list_windows.go returns items like "hiberfil.sys", "pagefile.sys" and "$Recycle.Bin".

## History

The name 'zero touch backup' is both historic and anamalous.

Zero-touch backup originated in mid-2000s as a Windows application that ran in the background and scanned all internal & external drives periodically (and external drives when they came online) for backup config files. These config files would then be run at preset times to update the backups. The best use of this tool was to create a backup configuration file on your backup drive, so that you just need to connect the drive to your system to update the backup! Some hard drive manufacturer (WD?) was advertising at that time that they had a button the drive and you just need to press that button to update the backup. They called it "one-touch backup". This tool need no 'touch' at all, since it updated the backup as soon as you connected the drive. Hence the name "zero-touch backup" (ZtB for short).

Fast forward to 2014 or so when I got sick of Microsoft operating systems and switched over entirely to Linux for all my personal needs. I tried using rsync and a few other backup tools but nothing worked as properly and cleanly (for me) as my beloved ZtB, so I rewrote a command line version of Zero-touch Backup in Linux and called it ztbackup. It would be more than 5 years before I finally uploaded the project to github.

With the dawn of 2020s, I happened to wander into Go language and decided to develop a Go version of ztbackup both as an exercise and as a way to make ztbackup more portable. Gozt is the result of that effort.

So, gozt continues to be called 'zero-touch backup' even though the concept of 'zero-touch' no longer applies to it!

## Final notes

Tests have shown that this GO language version of this application is somewhat slower than the C++ version (https://github.com/gkdada/ztbackup). However, the codebase of this GO version is smaller, and the general opinion is applications coded in GO are safer since it eliminates worst of the unforced errors like pointer overflow and such. 

Also, this GO version is more portable and hence, I have made effort to include customized system exclusion lists for all major operating systems. If you can add more system file names that need to be excluded (for smoother and trouble-free backup experience), please create pull requests for them.

Thanks for trying out this application.
