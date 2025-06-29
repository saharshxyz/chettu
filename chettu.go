package main

import (
	"bufio"
	"embed"
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	ignore "github.com/sabhiram/go-gitignore"
	"github.com/spf13/pflag"
	"golang.design/x/clipboard"
)

//go:embed project.tmpl
var templateFS embed.FS

type Config struct {
	IgnoreFiles  []string
	IgnoreLines  []string
	Directories  []string
	ResetIgnore  bool
	MaxCopySize  int64
	OutputFile   string
	ForceReplace bool
	ProjectName  string
}

type Project struct {
	XMLName     xml.Name `xml:"project"`
	ProjectName string   `xml:"project_name,attr,omitempty"`
	FileTree    string   `xml:"file_tree"`
	Files       []File   `xml:"file"`
}

type File struct {
	Path    string `xml:"file_path"`
	Content string `xml:"file_content"`
}

var defaultConfig = Config{
	IgnoreFiles:  []string{".gitignore", ".chettuignore"},
	IgnoreLines:  []string{".git", "*.otf", "*.jpeg", "*.jpg", "*.png", "*.gif", "*.bmp", "*.tif", "*.tiff", "*.webp", "*.avif", "*.heif", "*.heic", "*.psd", "*.psp", "*.xpm", "*.ppm", "*.pgm", "*.pbm", "*.hdr", "*.img", "*.ras", "*.ico", "*.cur", "*.dds", "*.svg", "*.ai", "*.eps", "*.pdf", "*.cdr", "*.wmf", "*.emf", "*.cr2", "*.nef", "*.arw", "*.orf", "*.raf", "*.rw2", "*.dng", "*.mp4", "*.avi", "*.mov", "*.wmv", "*.mkv", "*.flv", "*.webm", "*.mpg", "*.mpeg", "*.3gp", "*.ogv", "*.m4v", "*.asf", "*.apng", "*.mng", "*.ktx", "*.pvr", "*.astc", "*.gltf", "*.glb", "*.obj", "*.fbx", "*.stl", "*.dae", "*.usdz"}, // Mostly just image/video files that should always be ignored
	Directories:  []string{"./"},
	ResetIgnore:  false,
	MaxCopySize:  50000,
	OutputFile:   "",
	ForceReplace: false,
	ProjectName:  "",
}

func main() {
	config := parseFlags()
	ignored, config := setupProject(config)
	run(config, ignored)
}

func run(config Config, ignored *ignore.GitIgnore) {
	project := genProject(config.Directories, ignored, config.ProjectName)
	output := generateOutput(project)

	if config.OutputFile != "" {
		writeToFile(output, config.OutputFile, config.ForceReplace)
	}

	if config.MaxCopySize > 0 {
		copyToClipboard(output, config.MaxCopySize)
	}
}

func genProject(dirs []string, ignored *ignore.GitIgnore, projectName string) Project {
	var project Project
	project.ProjectName = projectName
	var filePaths []string

	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			handleError("Error walking file", err)

			if ignored.MatchesPath(path) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if !d.IsDir() {
				filePaths = append(filePaths, path)
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

	project.FileTree = generateFileTree(filePaths)
	return project
}

func generateFileTree(paths []string) string {
	tree := make(map[string]interface{})
	for _, path := range paths {
		parts := strings.Split(path, string(filepath.Separator))
		currentLevel := tree
		for i, part := range parts {
			if part == "." {
				continue
			}
			if i == len(parts)-1 {
				currentLevel[part] = nil
			} else {
				if _, ok := currentLevel[part]; !ok {
					currentLevel[part] = make(map[string]interface{})
				}
				currentLevel = currentLevel[part].(map[string]interface{})
			}
		}
	}

	var builder strings.Builder
	builder.WriteString(".\n")
	printTree(&builder, tree, "")
	return builder.String()
}

func printTree(builder *strings.Builder, tree map[string]interface{}, prefix string) {
	keys := make([]string, 0, len(tree))
	for k := range tree {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, key := range keys {
		isLast := i == len(keys)-1
		connector := "├── "
		newPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			newPrefix = prefix + "    "
		}

		builder.WriteString(prefix)
		builder.WriteString(connector)
		builder.WriteString(key)

		if subTree, ok := tree[key].(map[string]interface{}); ok {
			builder.WriteString("\n")
			printTree(builder, subTree, newPrefix)
		} else {
			builder.WriteString("\n")
		}
	}
}

func parseFlags() Config {
	ignoreLine := pflag.StringArrayP("ignore-line", "l", defaultConfig.IgnoreLines, "Append to ignore lines")
	ignoreFile := pflag.StringArrayP("ignore-file", "f", defaultConfig.IgnoreFiles, "Append to ignore files")
	directory := pflag.StringArrayP("directory", "d", defaultConfig.Directories, "Set directories")
	resetIgnore := pflag.Bool("reset-ignore", defaultConfig.ResetIgnore, "Reset ignore lists before appending")
	maxCopySize := pflag.Int64P("copy", "c", defaultConfig.MaxCopySize, "Enable clipboard copy with optional maximum size in characters")
	outputFile := pflag.StringP("output-file", "o", defaultConfig.OutputFile, "Specify the output file path")
	forceReplace := pflag.BoolP("output-file-replace", "R", defaultConfig.ForceReplace, "Force replacement of existing output file")
	projectName := pflag.StringP("project-name", "n", defaultConfig.ProjectName, "Set project name")

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
		ProjectName:  *projectName,
	}
}

func setupProject(config Config) (*ignore.GitIgnore, Config) {
	ignoreFiles, ignoreLines := processIgnoreFlags(config)
	ignored := compileIgnoreInMemory(ignoreFiles, ignoreLines)
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

func compileIgnoreInMemory(files, lines []string) *ignore.GitIgnore {
	var patterns []string

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			patterns = append(patterns, scanner.Text())
		}
		f.Close()
	}

	patterns = append(patterns, lines...)

	ignored := ignore.CompileIgnoreLines(patterns...)
	return ignored
}

func generateOutput(project Project) string {
	funcMap := template.FuncMap{
		"indent": func(content string) string {
			return indentContent(content, "\t")
		},
	}

	tmpl, err := templateFS.ReadFile("project.tmpl")
	handleError("Error reading embedded template file", err)

	t := template.Must(template.New("project").Funcs(funcMap).Parse(string(tmpl)))
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
		err := clipboard.Init()
		handleError("Error initializing clipboard", err)

		clipboard.Write(clipboard.FmtText, []byte(content))
		fmt.Printf("\nOutput(%d characters) copied to clipboard\n", contentSize)
	}
}

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
