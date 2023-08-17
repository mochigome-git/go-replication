package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	for {
		err := syncDirectory("G:/マイドライブ/オレンジ/2024年度", "C:/OneDrive/オレンジ/2024年度")
		if err != nil {
			fmt.Println(err)
		}

		removeFilesNotInSource("C:/OneDrive/オレンジ/2024年度", "G:/マイドライブ/オレンジ/2024年度")
		removeFilesManuallyDeleted("C:/OneDrive/オレンジ/2024年度", "G:/マイドライブ/オレンジ/2024年度")
		removeFilesNotInSource("G:/マイドライブ/オレンジ/2024年度", "C:/OneDrive/オレンジ/2024年度")
		removeFilesManuallyDeleted("G:/マイドライブ/オレンジ/2024年度", "C:/OneDrive/オレンジ/2024年度")

		time.Sleep(3 * time.Second) // Wait for 3 seconds before restarting the program
	}
}

func removeFilesNotInSource(dst, src string) error {
	dstFiles, err := filepath.Glob(filepath.Join(dst, "*"))
	if err != nil {
		return err
	}
	for _, dstFile := range dstFiles {
		srcFile := filepath.Join(src, filepath.Base(dstFile))
		if !fileExists(srcFile) && !fileExists(dstFile) && !isCreatedBySyncFile(dstFile) {
			err = os.Remove(dstFile)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func removeFilesManuallyDeleted(dst, src string) error {
	dstFiles, err := filepath.Glob(filepath.Join(dst, "*"))
	if err != nil {
		return err
	}

	for _, dstFile := range dstFiles {
		srcFile := filepath.Join(src, filepath.Base(dstFile))
		if !fileExists(srcFile) {
			err = os.Remove(dstFile)
			if err != nil {
				return err
			}
		}
	}

	srcFiles, err := filepath.Glob(filepath.Join(src, "*"))
	if err != nil {
		return err
	}

	for _, srcFile := range srcFiles {
		dstFile := filepath.Join(dst, filepath.Base(srcFile))
		if !fileExists(dstFile) {
			err = os.Remove(srcFile)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func syncDirectory(src, dst string) error {
	// Create the destination directory if it doesn't exist
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		err := os.MkdirAll(dst, 0755)
		if err != nil {
			return err
		}
	}
	// Get a list of all files and directories in the source directory
	files, err := filepath.Glob(filepath.Join(src, "*"))
	if err != nil {
		return err
	}

	// Clone or update files in the destination directory
	for _, file := range files {
		fi, err := os.Stat(file)
		if err != nil {
			return err
		}

		if fi.IsDir() {
			// If the file is a directory, call syncDirectory recursively
			err = syncDirectory(file, filepath.Join(dst, fi.Name()))
			if err != nil {
				return err
			}
		} else {
			// If the file is a regular file, copy it to the destination directory
			err = syncFile(file, filepath.Join(dst, fi.Name()))
			if err != nil {
				return err
			}
		}
	}

	// Sync files in the source directory with those in the destination directory
	dstFiles, err := filepath.Glob(filepath.Join(dst, "*"))
	if err != nil {
		return err
	}

	for _, dstFile := range dstFiles {
		srcFile := filepath.Join(src, filepath.Base(dstFile))

		// Check if the file in the destination directory was created by the syncFile function
		createdBySyncFile := false
		for _, file := range files {
			if filepath.Base(file) == filepath.Base(dstFile) {
				createdBySyncFile = true
				break
			}
		}

		if !createdBySyncFile {
			if !fileExists(srcFile) {
				// If the file does not exist in the source directory, copy it from the destination directory
				err = syncFile(dstFile, srcFile)
				if err != nil {
					return err
				}
			} else {
				// If the file exists in both the source and destination directories, copy it from the source directory
				err = syncFile(srcFile, dstFile)
				if err != nil {
					if os.IsNotExist(err) {
						// If the file does not exist in the source directory, remove it from the destination directory
						err = os.Remove(dstFile)
						if err != nil {
							return err
						}
					} else if strings.Contains(err.Error(), "created by SyncFile") {
						// If the error message contains the string "created by SyncFile", ignore the error
						continue
					} else {
						return err
					}
				}
			}
		}
	}

	return nil
}

func isCreatedBySyncFile(file string) bool {
	// Check if the file was created by the syncFile function
	if strings.HasSuffix(file, ".sync") {
		return true
	}

	return false
}

func syncFile(src, dst string) error {
	// Check if the destination file was created by the syncFile function
	if fileExists(dst) && isCreatedBySyncFile(dst) {
		return nil
	}
	// Open the source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy the contents of the source file to the destination file
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		// Ignore the error and continue if the file is a Google Sheet with an "Incorrect function" error
		if strings.Contains(err.Error(), "Incorrect function") {
			return nil
		}
		return err
	}

	// Set the modification time of the destination file to that of the source file
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime())
	if err != nil {
		return err
	}

	return nil
}

func fileExists(file string) bool {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return false
	} else if err != nil {
		return false
	}
	return true
}
