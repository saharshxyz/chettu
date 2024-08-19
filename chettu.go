package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
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
	ignorePaths        stringSet
	directories        []string
	ignoreFiles        []string
	maxClipboardSize   int
	copyToClipboard    bool
	outputFile         string
	forceOutputReplace bool
}

type stringSet map[string]struct{}

func (s stringSet) Add(value string)           { s[value] = struct{}{} }
func (s stringSet) Contains(value string) bool { _, ok := s[value]; return ok }
func (s stringSet) ToSlice() []string {
	result := make([]string, 0, len(s))
	for k := range s {
		result = append(result, k)
	}
	return result
}

func main() {
	cfg := parseFlags()
	setDefaultValues(&cfg)

	ignorePatterns, err := loadIgnorePatterns(cfg.ignoreFiles, cfg.ignorePaths)
	if err != nil {
		fmt.Printf("Error loading ignore patterns: %v\n", err)
		return
	}

	output := generateOutput(cfg, ignorePatterns)
	handleOutput(output, cfg)
}

func parseFlags() config {
	cfg := config{ignorePaths: make(stringSet)}

	flag.Func("i", "Paths to ignore (can be used multiple times). Use -i \"\" to clear default ignore patterns.", func(value string) error {
		cfg.ignorePaths.Add(value)
		return nil
	})
	flag.Var((*stringSliceFlag)(&cfg.directories), "d", "Directories to process (can be used multiple times)")
	flag.Var((*stringSliceFlag)(&cfg.ignoreFiles), "ignore-file", "Files containing ignore patterns (can be used multiple times, default: .gitignore). Use -ignore-file \"\" to prevent loading any ignore files.")
	flag.Func("c", "Enable clipboard copy with optional max size (default: 50000)", parseClipboardFlag(&cfg))
	flag.StringVar(&cfg.outputFile, "output-file", "", "Output file path")
	flag.BoolVar(&cfg.forceOutputReplace, "force-replace-output", false, "Force replace existing output file without prompting")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nUse -c without a value to copy with default max size (%d)\n", defaultMaxClipboardSize)
	}

	flag.Parse()

	if cfg.forceOutputReplace && cfg.outputFile == "" {
		fmt.Println("Error: -force-replace-output flag requires -output-file to be specified")
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}

func parseClipboardFlag(cfg *config) func(string) error {
	return func(flagValue string) error {
		cfg.copyToClipboard = true
		if flagValue == "" {
			cfg.maxClipboardSize = defaultMaxClipboardSize
			return nil
		}
		var err error
		cfg.maxClipboardSize, err = strconv.Atoi(flagValue)
		if err != nil {
			return fmt.Errorf("invalid value for -c: %v", err)
		}
		return nil
	}
}

func setDefaultValues(cfg *config) {
	if len(cfg.directories) == 0 {
		cfg.directories = []string{"."}
	}
	if len(cfg.ignoreFiles) == 0 {
		cfg.ignoreFiles = []string{defaultIgnoreFile}
	}
}

func loadIgnorePatterns(ignoreFiles []string, ignorePaths stringSet) ([]string, error) {
	patterns, err := loadIgnorePatternsFromFiles(ignoreFiles)
	if err != nil {
		return nil, err
	}
	return append(patterns, ignorePaths.ToSlice()...), nil
}

func generateOutput(cfg config, ignorePatterns []string) string {
	var output strings.Builder
	var filePaths []string

	output.WriteString("<documents>\n")

	for _, root := range cfg.directories {
		processDirectory(root, ignorePatterns, &output, &filePaths, os.Stdout)
	}

	output.WriteString("\n")

	for _, filePath := range filePaths {
		appendFileContents(filePath, &output)
	}

	output.WriteString("</documents>\n")

	return output.String()
}

func processDirectory(root string, ignorePatterns []string, output *strings.Builder, filePaths *[]string, treeOutput io.Writer) {
	ignoreParser := ignore.CompileIgnoreLines(ignorePatterns...)

	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Printf("Error getting absolute path for %s: %v\n", root, err)
		return
	}

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == absRoot {
			return err
		}

		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}

		fullRelPath := filepath.Join(filepath.Base(root), relPath)
		if ignoreParser.MatchesPath(fullRelPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		fmt.Fprintln(treeOutput, fullRelPath)
		output.WriteString(fullRelPath + "\n")

		if !info.IsDir() {
			*filePaths = append(*filePaths, fullRelPath)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path %v: %v\n", root, err)
	}

	fmt.Fprintln(treeOutput)
}

func loadIgnorePatternsFromFiles(ignoreFiles []string) ([]string, error) {
	var patterns []string
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current working directory: %v", err)
	}

	for _, ignoreFile := range ignoreFiles {
		ignorePath := filepath.Join(cwd, ignoreFile)
		file, err := os.Open(ignorePath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip if file doesn't exist
			}
			return nil, fmt.Errorf("error opening %s: %v", ignoreFile, err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				patterns = append(patterns, line)
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("error reading %s: %v", ignoreFile, err)
		}
	}
	return patterns, nil
}

func appendFileContents(filePath string, output *strings.Builder) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		fmt.Printf("Error getting absolute path for %s: %v\n", filePath, err)
		return
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		fmt.Printf("Error reading file %s: %v\n", filePath, err)
		return
	}

	output.WriteString("\t<document>\n")
	fmt.Fprintf(output, "\t\t<source>%s</source>\n", filePath)
	output.WriteString("\t\t<document_content>\n")
	output.Write(content)
	output.WriteString("\n\t\t</document_content>\n")
	output.WriteString("\t</document>\n")
}

func handleOutput(output string, cfg config) {
	if cfg.outputFile != "" {
		writeToFile(output, cfg)
	} else if !cfg.copyToClipboard {
		fmt.Print(output)
	}

	if cfg.copyToClipboard {
		copyToClipboard(output, cfg.maxClipboardSize)
	}
}

func writeToFile(output string, cfg config) {
	if !cfg.forceOutputReplace && fileExists(cfg.outputFile) {
		if !promptReplace(cfg.outputFile) {
			fmt.Println("Operation cancelled.")
			return
		}
	}

	err := os.WriteFile(cfg.outputFile, []byte(output), 0644)
	if err != nil {
		fmt.Printf("Error writing to output file: %v\n", err)
		return
	}

	fmt.Printf("\nOutput written to %s\n", cfg.outputFile)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func promptReplace(filePath string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("File %s already exists. Replace? (y/N): ", filePath)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func copyToClipboard(output string, maxSize int) {
	if len(output) > maxSize {
		fmt.Printf("Error: Output size (%d) exceeds the maximum clipboard size (%d)\n", len(output), maxSize)
		return
	}

	if err := clipboard.WriteAll(output); err != nil {
		fmt.Printf("Failed to copy to clipboard: %v\n", err)
		return
	}

	fmt.Printf("Output (%d characters) has been copied to clipboard.\n", len(output))
}

type stringSliceFlag []string

func (i *stringSliceFlag) String() string { return strings.Join(*i, ", ") }
func (i *stringSliceFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}
