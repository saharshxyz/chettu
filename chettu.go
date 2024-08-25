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

func compileIgnore(files []string, lines []string) *ignore.GitIgnore {
	tmpFile, err := os.CreateTemp("", ".ignore*")
	handleError("Error creating temporary ignore file", err)
	defer os.Remove(tmpFile.Name())

	var content []byte

	for _, file := range files {
		if fileContent, err := os.ReadFile(file); err != nil {
			fmt.Printf("Error reading file: %v\n", err)
		} else {
			content = append(content, fileContent...)
			content = append(content, '\n')
		}
	}

	for _, line := range lines {
		content = append(content, []byte(line+"\n")...)
	}

	for _, fileName := range files {
		content = append(content, []byte(fileName+"\n")...)
	}

	if err := os.WriteFile(tmpFile.Name(), content, 0644); err != nil {
		handleError("Error writing to temporary file", err)
	}

	if err := tmpFile.Close(); err != nil {
		handleError("Error closing temporary file", err)
	}

	ignored, err := ignore.CompileIgnoreFile(tmpFile.Name())
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
