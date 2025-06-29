# Chettu (చెట్టు) - Project File Collector

[![Go Report Card](https://goreportcard.com/badge/github.com/saharshxyz/chettu)](https://goreportcard.com/report/github.com/saharshxyz/chettu)

Chettu (Telugu for "tree") is a command-line tool that generates a comprehensive XML representation of your project's file structure and content. Perfect for providing context to LLMs, archiving project states, or programmatic codebase analysis.

## Features

-   **Recursive Directory Scanning** with customizable ignore rules via `.gitignore` and `.chettuignore`
-   **XML Output** containing both file tree visualization and complete file contents
-   **Flexible Command-Line Control** for directories, ignore patterns, and output options
-   **Clipboard Integration** with configurable size limits
-   **Safe File Handling** with overwrite confirmation prompts

## Installation

```shell
go install github.com/saharshxyz/chettu
```

## Quick Start

```shell
# Scan current directory
chettu

# Generate XML output file
chettu -o project_snapshot.xml

# Scan specific directory with custom ignores
chettu -d /path/to/project -l "*.log" -o output.xml
```

## Building from Source

```shell
git clone https://github.com/saharshxyz/chettu.git
cd chettu
go build -o chettu .
./chettu -o output.xml
```

## Command-line Flags

| Flag                    | Short | Description                                          | Default                           |
| ----------------------- | ----- | ---------------------------------------------------- | --------------------------------- |
| `--directory`           | `-d`  | Directory to scan (repeatable)                       | `./`                              |
| `--ignore-line`         | `-l`  | Pattern to ignore (repeatable)                       | `.git` + binary/media extensions  |
| `--ignore-file`         | `-f`  | Ignore file to use (repeatable)                      | `[".gitignore", ".chettuignore"]` |
| `--reset-ignore`        |       | Discard default ignore patterns                      | `false`                           |
| `--project-name`        | `-n`  | Set project name (appears in output)                 | `""` (no project name)            |
| `--output-file`         | `-o`  | Output file path                                     | `""` (no file output)             |
| `--output-file-replace` | `-R`  | Force overwrite existing output file (requires `-o`) | `false`                           |
| `--copy`                | `-c`  | Copy to clipboard with max size in bytes             | `50000` (50KB)                    |

## Usage Examples

```shell
# Multiple directories
chettu -d ./src -d ./docs

# Set project name
chettu -n "My Awesome Project"

# Custom ignore patterns
chettu -l "*.log" -l "dist"

# Custom ignore file
chettu -f .my_custom_ignores

# Only custom ignores (reset defaults)
chettu --reset-ignore -l "*.tmp" -l "vendor"

# Force file overwrite
chettu -o project_structure.xml -R

# Combined options
chettu -n "My Project" -d /path/to/project -l "*.log" -o output.xml -c 200000
```

## Output Format

Generates XML with `<project>` root containing:
- `<project_name>`: Project name (when specified with `-n` flag)
- `<source_tree>`: Text-based directory visualization
- `<files>`: Individual `<file>` elements with `path` attribute and content

```xml
<project>
    <project_name>My Awesome Project</project_name>
    <source_tree>
        .
        ├── README.md
        ├── src
        │   └── main.go
        └── docs
            └── guide.md
    </source_tree>
    <files>
        <file path="README.md">
            # My Project
            This is the main README file.
        </file>
        <file path="src/main.go">
            package main

            import "fmt"

            func main() {
                fmt.Println("Hello, World!")
            }
        </file>
        <file path="docs/guide.md">
            ## Guide
            This is a guide for using My Project.
        </file>
    </files>
</project>
```

## Dependencies

- [github.com/spf13/pflag](https://github.com/spf13/pflag)
- [github.com/sabhiram/go-gitignore](https://github.com/sabhiram/go-gitignore)
- [golang.design/x/clipboard](https://pkg.go.dev/golang.design/x/clipboard)