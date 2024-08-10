package go_fileutils_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	gfu "github.com/rocco-gossmann/go_fileutils"
	"github.com/rocco-gossmann/go_utils"
	"github.com/stretchr/testify/assert"
)

const (
	DEMOFILEA = "./demoFileA.txt"
	DEMOFILEB = "./demoFileB.txt"
)

func createDemoFile(t *testing.T, file string) {

	go_utils.MkDir(filepath.Dir(file))

	demoFileA, err := os.Create(file)
	assert.Nil(t, err, "coult not create "+file)
	demoFileA.Write([]byte("DEMO FILE CONTENT"))
	demoFileA.Close()
}

func timeoutContext(t *testing.T, timeout time.Duration, fnc func(context.Context, chan<- struct{})) {

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	doneChan := make(chan struct{})

	go fnc(ctx, doneChan)

	// Waiting for either ctx or done
	for {
		select {
		case <-ctx.Done():
			t.Log("hit timeout")
			t.FailNow()
			cancel()
			return

		case <-doneChan:
			t.Log("done before timeout")
			cancel()
			return
		}
	}
}

func TestCopyWithProgress(t *testing.T) {

	os.Remove(DEMOFILEA)
	os.Remove(DEMOFILEB)

	createDemoFile(t, "./"+DEMOFILEA)
	defer os.Remove("./" + DEMOFILEA)

	t.Run("Test if copy even starts", func(t *testing.T) {

		timeoutContext(t, 2*time.Second, func(ctx context.Context, done chan<- struct{}) {

			defer os.Remove(DEMOFILEB)

			progressChan := gfu.CopyFile(DEMOFILEA, DEMOFILEB)

			progress := <-progressChan
			assert.Equal(t, progress.State, gfu.STATE_START_FILE, "expected state to be STATE_START_FILE")
			done <- struct{}{}

		})
	})

	t.Run("test copy from a to b", func(t *testing.T) {
		timeoutContext(t, 2*time.Second, func(ctx context.Context, c chan<- struct{}) {

			// Keep track of states, that came in
			states := make([]gfu.ProgressState, 0, 5)

			// Start copying
			progressChan := gfu.CopyFile(DEMOFILEA, DEMOFILEB)
			defer os.Remove(DEMOFILEB)

			// handle Copy progression
			count := 0

		listener:
			for count < 20 {
				progress := <-progressChan
				states = append(states, progress.State)

				switch progress.State {
				case gfu.STATE_ERROR:
					fmt.Println(progress.Error)
					fallthrough

				case gfu.STATE_FINISHED:
					t.Log("State STATE_FINISHED should not happen on single file copy")
					t.FailNow()
					fallthrough

				case gfu.STATE_END_FILE:
					assert.Nil(t, progress.Error, "process did not go without errors")
					break listener

				}

				count += 1
			}

			// check results
			assert.Less(t, count, 20, "never received a STATE_FINSIED or STATE_ERROR")
			assert.Greater(t, len(states), 1, "There should be at least 2 states cought")
			assert.Equal(t, states[0], gfu.STATE_START_FILE, "First state should have been 'STATE_START_FILE'")
			assert.Equal(t, states[len(states)-1], gfu.STATE_END_FILE, "Last state should have been 'STATE_END_FILE'")

			c <- struct{}{}

		})
	})

	t.Run("test copy batch function", func(t *testing.T) {

		go_utils.MkDir("./testsrc")
		createDemoFile(t, "./testsrc/demo1.txt")
		createDemoFile(t, "./testsrc/demo2.txt")
		createDemoFile(t, "./testsrc/demo3.txt")

		timeoutContext(t, 2*time.Second, func(ctx context.Context, c chan<- struct{}) {

			filesCopied := make(map[string]struct{})
			copiesStarted := 0
			progressChan := gfu.CopyRecursive("./testsrc", "./testtar", "")

		wait_loop:
			for {
				progress := <-progressChan

				switch progress.State {

				case gfu.STATE_START_FILE:
					copiesStarted += 1

				case gfu.STATE_END_FILE:
					fmt.Println(progress)
					filesCopied[progress.CurrentSource] = struct{}{}

				case gfu.STATE_FINISHED:
					break wait_loop

				}
			}

			assert.Equal(t, 3, copiesStarted, "there should have been 3 copies started")

			var assertCopied = func(tar string) {

				_, err := os.Stat(tar)
				assert.Nilf(t, err, "Error while checking if file '%s' exists", tar)
			}

			fmt.Println(filesCopied)

			assertCopied("./testtar/demo3.txt")
			assertCopied("./testtar/demo2.txt")
			assertCopied("./testtar/demo1.txt")

			c <- struct{}{}
		})

		os.RemoveAll("./testsrc")
		os.RemoveAll("./testtar")

	})
}
