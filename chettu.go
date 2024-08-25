package main

import (
	"fmt"
	"os"
	"path/filepath"

	ignore "github.com/sabhiram/go-gitignore"
)

var (
	ignoreFiles = []string{".gitignore", ".chettuignore"}
	ignoreLines = []string{".git"}
	dirs        = []string{"./"}
	ignored     *ignore.GitIgnore
)

func init() {
	fmt.Println(("Running chettu"))
}

func main() {

	ignored = compileIgnore(ignoreFiles, ignoreLines)

	for _, dir := range dirs {
		printDir(dir)
	}
}

func handleError(message string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
		os.Exit(1)
	}
}

func compileIgnore(files, lines []string) *ignore.GitIgnore {
	createIgnoreFile := func(files, lines []string) (string, func()) {
		tmpFile, err := os.CreateTemp("", ".tmpChettuIgnore*")
		handleError("Error creating temporary ignore file", err)

		cleanup := func() {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
		}

		var content []byte

		for _, file := range files {
			if fileContent, err := os.ReadFile(file); err == nil {
				content = append(content, []byte("#"+file+"\n")...)
				content = append(content, fileContent...)
				content = append(content, '\n', '\n')
			}
		}

		content = append(content, []byte("# ignore lines"+"\n")...)
		for _, line := range append(lines, files...) {
			content = append(content, []byte(line+"\n")...)
		}

		handleError("Error writing to temporary file", os.WriteFile(tmpFile.Name(), content, 0644))
		handleError("Error closing temporary file", tmpFile.Close())

		return tmpFile.Name(), cleanup
	}

	fileName, tmpIgnoreFileCleanup := createIgnoreFile(files, lines)
	defer tmpIgnoreFileCleanup()

	ignored, err := ignore.CompileIgnoreFile(fileName)
	handleError("Unable to compile ignore file", err)

	return ignored
}

func printDir(dir string) {
	isIgnored := func(path string) bool {
		getAbsolutePath := func(path string) string {
			absPath, err := filepath.Abs(path)
			handleError("Error getting absolute filepath", err)

			return absPath
		}

		return ignored.MatchesPath(getAbsolutePath(path))
	}

	entries, err := os.ReadDir(dir)
	handleError("Error reading directory", err)
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		if !isIgnored(path) {
			println(path)

			if entry.IsDir() {
				printDir(path)
			}
		}
	}
}
