package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	ignore "github.com/sabhiram/go-gitignore"
)

const (
	defaultMaxClipboardSize = 500000
	defaultIgnoreFile       = ".gitignore"
)

type config struct {
	ignorePaths      stringSet
	directories      []string
	ignoreFiles      []string
	maxClipboardSize int
	copyToClipboard  bool
}

type stringSet map[string]struct{}

func (s stringSet) Add(value string) {
	s[value] = struct{}{}
}

func (s stringSet) Contains(value string) bool {
	_, ok := s[value]
	return ok
}

func (s stringSet) ToSlice() []string {
	result := make([]string, 0, len(s))
	for k := range s {
		result = append(result, k)
	}
	return result
}

func main() {
	cfg := parseFlags()

	if len(cfg.directories) == 0 {
		cfg.directories = append(cfg.directories, ".")
	}

	formattedOutput := formatOutput(cfg)

	fmt.Print(formattedOutput)

	handleClipboardCopy(formattedOutput, cfg)
}

func parseFlags() config {
	cfg := config{
		ignorePaths: make(stringSet),
	}

	flag.Func("i", "Paths to ignore (can be used multiple times). Use -i \"\" to clear default ignore patterns.", func(value string) error {
		cfg.ignorePaths.Add(value)
		return nil
	})
	flag.Var((*stringSliceFlag)(&cfg.directories), "d", "Directories to process (can be used multiple times)")
	flag.Var((*stringSliceFlag)(&cfg.ignoreFiles), "ignore-file", "Files containing ignore patterns (can be used multiple times, default: .gitignore). Use -ignore-file \"\" to prevent loading any ignore files.")

	flag.Func("c", "Enable clipboard copy with optional max size (default: 50000)", func(flagValue string) error {
		cfg.copyToClipboard = true
		if flagValue == "" {
			cfg.maxClipboardSize = defaultMaxClipboardSize
		} else {
			var err error
			cfg.maxClipboardSize, err = strconv.Atoi(flagValue)
			if err != nil {
				return fmt.Errorf("invalid value for -c: %v", err)
			}
		}
		return nil
	})

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nUse -c without a value to copy with default max size (%d)\n", defaultMaxClipboardSize)
	}

	flag.Parse()

	return cfg
}

func handleClipboardCopy(output string, cfg config) {
	if !cfg.copyToClipboard {
		return
	}

	if len(output) > cfg.maxClipboardSize {
		fmt.Printf("Error: Output size (%d) exceeds the maximum clipboard size (%d)\n", len(output), cfg.maxClipboardSize)
		return
	}

	if err := clipboard.WriteAll(output); err != nil {
		fmt.Printf("Failed to copy to clipboard: %v\n", err)
		return
	}

	fmt.Println("Output has been copied to clipboard.")
}

func formatOutput(cfg config) string {
	var output strings.Builder
	var filePaths []string

	output.WriteString("<documents>\n")

	for _, root := range cfg.directories {
		processDirectory(root, cfg.ignorePaths.ToSlice(), cfg.ignoreFiles, &output, &filePaths)
	}

	output.WriteString("\n")

	for _, filePath := range filePaths {
		printFileContents(filePath, &output)
	}

	output.WriteString("</documents>")

	return output.String()
}

func processDirectory(root string, ignorePatterns []string, ignoreFiles []string, output *strings.Builder, filePaths *[]string) {
	if len(ignoreFiles) == 0 {
		ignoreFiles = append(ignoreFiles, defaultIgnoreFile)
	}

	ignorePatterns = appendIgnorePatternsFromFiles(root, ignoreFiles, ignorePatterns)
	ignoreParser := ignore.CompileIgnoreLines(ignorePatterns...)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == root {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if ignoreParser.MatchesPath(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		path = ensureRelativePath(path)
		output.WriteString(path + "\n")
		if !info.IsDir() {
			*filePaths = append(*filePaths, path)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %v: %v\n", root, err)
	}
}

func appendIgnorePatternsFromFiles(root string, ignoreFiles, ignorePatterns []string) []string {
	for _, ignoreFile := range ignoreFiles {
		ignorePath := filepath.Join(root, ignoreFile)
		if _, err := os.Stat(ignorePath); err != nil {
			continue
		}

		file, err := os.Open(ignorePath)
		if err != nil {
			fmt.Printf("Error opening %s in %s: %v\n", ignoreFile, root, err)
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				ignorePatterns = append(ignorePatterns, line)
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading %s in %s: %v\n", ignoreFile, root, err)
		}
	}
	return ignorePatterns
}

func ensureRelativePath(path string) string {
	if !strings.HasPrefix(path, "./") {
		return "./" + path
	}
	return path
}

func printFileContents(filePath string, output *strings.Builder) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file %s: %v\n", filePath, err)
		return
	}

	output.WriteString("\t<document>\n")
	output.WriteString("\t\t<source>" + filePath + "</source>\n")
	output.WriteString("\t\t<document_content>\n")
	output.Write(content)
	output.WriteString("\n\t\t</document_content>\n")
	output.WriteString("\t</document>\n")
}

type stringSliceFlag []string

func (i *stringSliceFlag) String() string {
	return strings.Join(*i, ", ")
}

func (i *stringSliceFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}
