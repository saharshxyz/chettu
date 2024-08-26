package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/atotto/clipboard"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/spf13/pflag"
)

// Types
type Config struct {
	IgnoreFiles  []string
	IgnoreLines  []string
	Directories  []string
	ResetIgnore  bool
	MaxCopySize  int64
	OutputFile   string
	ForceReplace bool
}

type Project struct {
	XMLName  xml.Name `xml:"project"`
	FileTree []string `xml:"file_tree>file_path"`
	Files    []File   `xml:"file"`
}

type File struct {
	Path    string `xml:"file_path"`
	Content string `xml:"file_content"`
}

// Constants
var defaultConfig = Config{
	IgnoreFiles:  []string{".gitignore", ".chettuignore"},
	IgnoreLines:  []string{".git"},
	Directories:  []string{"./"},
	ResetIgnore:  false,
	MaxCopySize:  50000,
	OutputFile:   "",
	ForceReplace: false,
}

// Main function

func main() {
	config := parseFlags()
	ignored, config := setupProject(config)
	run(config, ignored)
}

// Core functionality
func run(config Config, ignored *ignore.GitIgnore) {
	project := genProject(config.Directories, ignored)
	output := generateOutput(project)

	if config.OutputFile != "" {
		writeToFile(output, config.OutputFile, config.ForceReplace)
	}

	if config.MaxCopySize > 0 {
		copyToClipboard(output, config.MaxCopySize)
	}
}

func genProject(dirs []string, ignored *ignore.GitIgnore) Project {
	var project Project

	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			handleError("Error walking file", err)

			if ignored.MatchesPath(path) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if !info.IsDir() {
				project.FileTree = append(project.FileTree, path)
				fmt.Println(path)

				content, err := os.ReadFile(path)
				handleError("Error reading file", err)

				project.Files = append(project.Files, File{
					Path:    path,
					Content: string(content),
				})
			}

			return nil
		})
		handleError("Error walking directory", err)
	}

	return project
}

// Configuration and setup
func parseFlags() Config {
	ignoreLine := pflag.StringArrayP("ignore-line", "l", defaultConfig.IgnoreLines, "Append to ignore lines")
	ignoreFile := pflag.StringArrayP("ignore-file", "f", defaultConfig.IgnoreFiles, "Append to ignore files")
	directory := pflag.StringArrayP("directory", "d", defaultConfig.Directories, "Set directories")
	resetIgnore := pflag.Bool("reset-ignore", defaultConfig.ResetIgnore, "Reset ignore lists before appending")
	maxCopySize := pflag.Int64P("copy", "c", defaultConfig.MaxCopySize, "Enable clipboard copy with optional maximum size")
	outputFile := pflag.StringP("output-file", "o", defaultConfig.OutputFile, "Specify the output file path")
	forceReplace := pflag.BoolP("output-file-replace", "R", defaultConfig.ForceReplace, "Force replacement of existing output file")

	pflag.Parse()

	if *forceReplace && *outputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: The -R (force replace) flag requires the -o (output file) flag to be specified.")
		os.Exit(1)
	}

	return Config{
		IgnoreFiles:  *ignoreFile,
		IgnoreLines:  *ignoreLine,
		Directories:  *directory,
		ResetIgnore:  *resetIgnore,
		MaxCopySize:  *maxCopySize,
		OutputFile:   *outputFile,
		ForceReplace: *forceReplace,
	}
}

func setupProject(config Config) (*ignore.GitIgnore, Config) {
	ignoreFiles, ignoreLines := processIgnoreFlags(config)
	ignored := compileIgnore(ignoreFiles, ignoreLines)
	return ignored, config
}

func processIgnoreFlags(config Config) ([]string, []string) {
	var ignoreFiles, ignoreLines []string

	if !config.ResetIgnore {
		ignoreFiles = defaultConfig.IgnoreFiles
		ignoreLines = defaultConfig.IgnoreLines
	}

	ignoreLines = append(ignoreLines, config.IgnoreLines...)
	ignoreFiles = append(ignoreFiles, config.IgnoreFiles...)

	return ignoreFiles, ignoreLines
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

// Output generation
func generateOutput(project Project) string {
	funcMap := template.FuncMap{
		"indent": func(content string) string {
			return indentContent(content, "\t\t")
		},
	}

	tmpl := `<project>
<file_tree>
	{{- range .FileTree}}
	<file_path>{{.}}</file_path>
	{{- end}}
</file_tree>
{{- range .Files}}
<file>
	<file_path>{{.Path}}</file_path>
	<file_content>
{{indent .Content}}
	</file_content>
</file>
{{- end}}
</project>`

	t := template.Must(template.New("project").Funcs(funcMap).Parse(tmpl))
	var buffer strings.Builder
	if err := t.Execute(&buffer, project); err != nil {
		handleError("Error executing template", err)
	}
	return buffer.String()
}

func indentContent(content, indent string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}

// File operations
func writeToFile(content, filePath string, forceReplace bool) {
	if !forceReplace && fileExists(filePath) {
		fmt.Printf("File %s already exists. Overwrite? (y/N): ", filePath)
		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) != "y" {
			fmt.Println("Operation cancelled.")
			return
		}
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	handleError("Error writing to file", err)

	fmt.Printf("Output written to %s\n", filePath)
}

func copyToClipboard(content string, maxCopySize int64) {
	if contentSize := int64(len(content)); contentSize > maxCopySize {
		fmt.Fprintf(os.Stderr, "\nError: content size (%d) is greater than max copy size (%d)\n", contentSize, maxCopySize)
	} else {
		err := clipboard.WriteAll(content)
		handleError("Error copying to clipboard", err)

		fmt.Printf("\nOutput(%d) copied to clipboard\n", contentSize)
	}
}

// Utility functions
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func handleError(message string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s: %v\n", message, err)
		os.Exit(1)
	}
}
