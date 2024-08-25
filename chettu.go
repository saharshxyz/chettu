package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	ignore "github.com/sabhiram/go-gitignore"
)

var (
	ignoreFiles = []string{".gitignore", ".chettuignore"}
	ignoreLines = []string{".git"}
	dirs        = []string{"./"}
)

type Project struct {
	XMLName  xml.Name `xml:"project"`
	FileTree []string `xml:"file_tree>file_path"`
	Files    []File   `xml:"file"`
}

type File struct {
	Path    string `xml:"file_path"`
	Content string `xml:"file_content"`
}

func init() {
	fmt.Println(("Running chettu"))
}

func main() {
	ignored := compileIgnore(ignoreFiles, ignoreLines)

	project := genProject(dirs, ignored)

	printProject(project)

}

func printProject(project Project) {
	indentContent := func(content, indent string) string {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if line != "" {
				lines[i] = indent + line
			}
		}
		return strings.Join(lines, "\n")
	}

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
	if err := t.Execute(os.Stdout, project); err != nil {
		fmt.Println("Error executing template:", err)
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
