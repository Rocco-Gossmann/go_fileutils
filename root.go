package go_fileutils

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/rocco-gossmann/go_utils"
)

func NewProgressChannel() chan BatchProgress {
	return make(chan BatchProgress)
}

func CopyFile(from, to string) <-chan BatchProgress {
	progress := NewProgressChannel()
	go copyFile(from, to, progress)
	return progress
}

func submitErr(err error, progressChan chan<- BatchProgress) {
	progressChan <- BatchProgress{State: STATE_ERROR, Error: err}
}

func copyFile(from, to string, progressChan chan<- BatchProgress) {

	progress := BatchProgress{}

	var (
		srcStat    fs.FileInfo
		err        error
		fFrom, fTo *os.File
	)

	if srcStat, err = os.Stat(from); err != nil {
		submitErr(err, progressChan)
		return
	}

	if srcStat.IsDir() {
		submitErr(errors.New("'go_utils.copyFile' can't copy Directories"), progressChan)
		return
	}

	if fFrom, err = os.Open(from); err != nil {
		submitErr(err, progressChan)
		return
	}
	defer fFrom.Close()

	if fTo, err = os.OpenFile(to, os.O_WRONLY|os.O_CREATE, srcStat.Mode().Perm()); err != nil {
		submitErr(err, progressChan)
		return
	}
	defer fTo.Close()

	progress.State = STATE_START_FILE
	progress.CurrentSource = from
	progress.CurrentTarget = to
	progress.BytesTotal = int(srcStat.Size())
	progress.BytesCopied = 0

	progressChan <- progress

	_, err = go_utils.CopyWithProgress(fFrom, fTo, func(bytesCopied int) {
		progress.State = STATE_COPY
		progress.BytesCopied = bytesCopied
		progressChan <- progress
	})

	progress.State = STATE_END_FILE
	progress.BytesCopied = progress.BytesTotal
	progressChan <- progress

}

func CopyRecursive(root string, dst string, cutoffPath string) <-chan BatchProgress {

	progressChannel := NewProgressChannel()

	go func() {
		root, err := filepath.Abs(root)

		if err != nil {
			submitErr(err, progressChannel)
			return
		}

		var cutRoot = fmt.Sprintf("%s%c%s", root, os.PathSeparator, cutoffPath)

		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {

			// DONE: Don't do anything if you are curretly processing a subdir
			if d.IsDir() {
				return nil
			}

			// DONE: Define destination Path
			cutPath, _ := strings.CutPrefix(path, cutRoot)
			dstPath := fmt.Sprintf("%s%c%s", dst, os.PathSeparator, cutPath)
			dstDir := filepath.Dir(dstPath)

			// DONE: make dir if not exists
			cutPath, _ = strings.CutPrefix(path, cutRoot)
			if err = go_utils.MkDir(dstDir); err != nil {
				submitErr(err, progressChannel)
			}

			// DONE: copy the file over
			copyFile(path, dstPath, progressChannel)

			return nil
		})

		// DONE: Inform client, that copy was finished
		progressChannel <- BatchProgress{
			State: STATE_FINISHED,
		}
	}()

	return progressChannel
}
