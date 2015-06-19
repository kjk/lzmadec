package lzmadec

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	timeLayout = "2006-01-02 15:04:05"
)

type detectionStateOf7z int

const (
	cmdNotChecked detectionStateOf7z = iota
	cmdPresent
	cmdNotPresent
)

var (
	// Err7zNotAvailable is returned if 7z executable is not available
	Err7zNotAvailable = errors.New("7z executable not available")

	// ErrNoEntries is returned if the archive has no files
	ErrNoEntries = errors.New("no entries in 7z file")

	errUnexpectedLines = errors.New("unexpected number of lines")

	mu  sync.Mutex
	d7z detectionStateOf7z
)

// Archive describes a single .7z archive
type Archive struct {
	Path    string
	Entries []Entry
}

// Entry describes a single file inside .7z archive
type Entry struct {
	Path       string
	Size       int
	PackedSize int // -1 means "size unknown"
	Modified   time.Time
	Attributes string
	CRC        string
	Encrypted  string
	Method     string
	Block      int
}

func detect7zCached() error {
	mu.Lock()
	defer mu.Unlock()
	if d7z == cmdNotChecked {
		if _, err := exec.LookPath("7z"); err != nil {
			d7z = cmdNotPresent
		} else {
			d7z = cmdPresent
		}
	}
	if d7z == cmdPresent {
		// checked and present
		return nil
	}
	// checked and not present
	return Err7zNotAvailable
}

/*
----------
Path = Badges.xml
Size = 4065633
Packed Size = 18990516
Modified = 2015-03-09 14:30:49
Attributes = ....A
CRC = 2C468F32
Encrypted = -
Method = BZip2
Block = 0
*/
func advanceToFirstEntry(scanner *bufio.Scanner) error {
	for scanner.Scan() {
		s := scanner.Text()
		if s == "----------" {
			return nil
		}
	}
	err := scanner.Err()
	if err == nil {
		err = ErrNoEntries
	}
	return err
}

func getEntryLines(scanner *bufio.Scanner) ([]string, error) {
	var res []string
	for scanner.Scan() {
		s := scanner.Text()
		s = strings.TrimSpace(s)
		if s == "" {
			break
		}
		res = append(res, s)
	}
	err := scanner.Err()
	if err != nil {
		return nil, err
	}
	if len(res) == 9 || len(res) == 0 {
		return res, nil
	}
	return nil, errUnexpectedLines
}

func parseEntryLines(lines []string) (Entry, error) {
	var e Entry
	var err error
	for _, s := range lines {
		parts := strings.SplitN(s, " =", 2)
		if len(parts) != 2 {
			return e, fmt.Errorf("unexpected line: '%s'", s)
		}
		name := strings.ToLower(parts[0])
		v := strings.TrimSpace(parts[1])
		switch name {
		case "path":
			e.Path = v
		case "size":
			e.Size, err = strconv.Atoi(v)
		case "packed size":
			e.PackedSize = -1
			if v != "" {
				e.PackedSize, err = strconv.Atoi(v)
			}
		case "modified":
			e.Modified, err = time.Parse(timeLayout, v)
		case "attributes":
			e.Attributes = v
		case "crc":
			e.CRC = v
		case "encrypted":
			e.Encrypted = v
		case "method":
			e.Method = v
		case "block":
			if v != "" {
				e.Block, err = strconv.Atoi(v)
			}
		default:
			err = fmt.Errorf("unexpected entry line '%s'", name)
		}
		if err != nil {
			return e, err
		}
	}
	return e, nil
}

func parse7zListOutput(d []byte) ([]Entry, error) {
	var res []Entry
	r := bytes.NewBuffer(d)
	scanner := bufio.NewScanner(r)
	err := advanceToFirstEntry(scanner)
	if err != nil {
		return nil, err
	}
	for {
		lines, err := getEntryLines(scanner)
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			// last entry
			break
		}
		e, err := parseEntryLines(lines)
		if err != nil {
			return nil, err
		}
		res = append(res, e)
	}
	return res, nil
}

// NewArchive uses 7z to extract a list of files in .7z archive
func NewArchive(path string) (*Archive, error) {
	err := detect7zCached()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("7z", "l", "-slt", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	entries, err := parse7zListOutput(out)
	if err != nil {
		return nil, err
	}
	return &Archive{
		Path:    path,
		Entries: entries,
	}, nil
}

type readCloser struct {
	rc  io.ReadCloser
	cmd *exec.Cmd
}

func (rc *readCloser) Read(p []byte) (int, error) {
	return rc.rc.Read(p)
}

func (rc *readCloser) Close() error {
	// if we want to finish before reading all the data, we need to close
	// it all the data has already been read, or else rc.cmd.Wait() will hang
	// if it's already closed then Close() will return 'invalid argument',
	// which we can ignore
	rc.rc.Close()
	return rc.cmd.Wait()
}

// GetFileReader returns a reader for reading a given file
func (a *Archive) GetFileReader(index int) (io.ReadCloser, error) {
	cmd := exec.Command("7z", "x", "-so", a.Path, a.Entries[index].Path)
	stdout, err := cmd.StdoutPipe()
	rc := &readCloser{
		rc:  stdout,
		cmd: cmd,
	}
	err = cmd.Start()
	if err != nil {
		stdout.Close()
		return nil, err
	}
	return rc, nil
}
