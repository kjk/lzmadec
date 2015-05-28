package lzmadec

import (
	"errors"
	"os/exec"
	"sync"
)

var (
	// Err7zNotAvailable is returned if 7z executable is not available
	Err7zNotAvailable  = errors.New("7z executable not available")
	detectionStateOf7z int // 0 - not checked, 1 - checked and present, 2 - checked and not present
	mu                 sync.Mutex
)

// Archive describes a single .7z archive
type Archive struct {
	Entries []*Entry
}

// Entry describes a single file inside .7z archive
type Entry struct {
}

func detect7zCached() error {
	mu.Lock()
	defer mu.Unlock()
	if detectionStateOf7z == 0 {
		cmd := exec.Command("7z")
		_, err := cmd.CombinedOutput()
		if err != nil {
			detectionStateOf7z = 2
		} else {
			detectionStateOf7z = 1
		}
	}
	if detectionStateOf7z == 1 {
		// checked and present
		return nil
	}
	// checked and not present
	return Err7zNotAvailable
}

// NewArchive uses 7z to extract a list of files in .7z archive
func NewArchive(path string) (*Archive, error) {
	err := detect7zCached()
	if err != nil {
		return nil, err
	}

	return nil, nil
}
