package renderer

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

var ppIncludeRe = regexp.MustCompile(`(?im)^#pragma\s+use\s+"([^"]+)"$`)

// Source represents a single source file.
type Source interface {
	// Contents reads the contents of the source file.
	Contents() ([]byte, error)
	// Dir returns the parent directory the file is located in.
	Dir() string
}

// SourceBuf is an implementation of the Source interface that keeps its
// contents in memory.
type SourceBuf string

// Contents implemetns the Source interface.
func (s SourceBuf) Contents() ([]byte, error) {
	return []byte(s), nil
}

// Dir implemetns the Source interface.
//
// A Sourcebuf has no parent directory, so the current working directory is
// returned instead.
func (s SourceBuf) Dir() string {
	return "."
}

// SourceFile is an implementation of the Source interface for real files.
type SourceFile struct {
	Filename string
}

// Includes recursively resolves dependencies in the specified file.
//
// The argument file is returned included in the returned list of files.
func Includes(filenames ...string) ([]SourceFile, error) {
	return processRecursive(filenames, []SourceFile{})
}

// Contents implemetns the Source interface.
func (s SourceFile) Contents() ([]byte, error) {
	fd, err := os.Open(s.Filename)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return ioutil.ReadAll(fd)
}

// Dir implemetns the Source interface.
func (s SourceFile) Dir() string {
	return filepath.Dir(s.Filename)
}

func processRecursive(filenames []string, sources []SourceFile) ([]SourceFile, error) {
	for _, filename := range filenames {
		absFilename, err := filepath.Abs(filename)
		if err != nil {
			return nil, err
		}
		currentFile := SourceFile{Filename: absFilename}
		shaderSource, err := currentFile.Contents()
		if err != nil {
			return nil, err
		}

		// We need to check for recursion using a set that includes the current
		// file. But we need to append the current file after all included sources
		// in the list of files. Create a new temporary set of included source
		// files for the recursion check.
		checkset := append(sources, currentFile)

		// Check for files being included in the current file so we can later
		// recurse into all of them.
		includeMatches := ppIncludeRe.FindAllSubmatch(shaderSource, -1)
		includes := make([]string, 0, len(includeMatches))
	outer:
		for _, submatch := range includeMatches {
			includedFile := string(submatch[1])
			if !filepath.IsAbs(includedFile) {
				includedFile = filepath.Join(filepath.Dir(absFilename), includedFile)
			} else {
				includedFile = filepath.Clean(includedFile)
			}

			// Check whether we have already included the referred file. This stops
			// infinite recursions.
			for _, inc := range checkset {
				if inc.Filename == includedFile {
					continue outer
				}
			}
			includes = append(includes, includedFile)
		}

		sources, err = processRecursive(includes, sources)
		if err != nil {
			return nil, err
		}
		sources = append(sources, currentFile)
	}

	return sources, nil
}
