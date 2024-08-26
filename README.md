# Chettu (చెట్టు) - Project File Collector

A Go-based command-line tool that generates an XML representation of your project's file structure and contents.

## Features

- Recursively scans specified directories
- Generates an XML-like output containing file paths and contents
- Supports custom ignore rules (based on `.gitignore`)
- Can copy output to clipboard (with size limit)
- Can write output to a file
- Customizable through command-line flags

## Installation

To install Project File Collector, make sure you have Go installed on your system, then run:

```
go get github.com/saharshxyz/chettu
```

## Usage

### Flags

Chettu supports the following command-line flags:

- `-l, --ignore-line <pattern>`: 
  - Appends the specified pattern to the ignore lines.
  - Can be used multiple times to add multiple patterns.
  - Default: `.git`

- `-f, --ignore-file <file>`: 
  - Appends the specified file to the list of ignore files.
  - Can be used multiple times to add multiple files.
  - Default: `[".gitignore", ".chettuignore"]`

- `-d, --directory <path>`: 
  - Sets the directories to scan.
  - Can be used multiple times to specify multiple directories.
  - Default: `./` (current directory)

- `--reset-ignore`: 
  - Resets the ignore lists before appending new ignore patterns or files.
  - Default: `false`

- `-c, --copy <size>`: 
  - Enables clipboard copy with an optional maximum size (in bytes).
  - If the output exceeds this size, it will not be copied to the clipboard.
  - Default: 50000 (50KB)

- `-o, --output-file <path>`: 
  - Specifies the output file path to write the XML content.
  - If not provided, output is not written to a file.
  - Default: "" (empty string, no file output)

- `-R, --output-file-replace`: 
  - Forces replacement of the existing output file without prompting.
  - Only applicable when `-o` flag is used.
  - Default: `false`

### Examples

1. Basic usage (scan current directory):
   ```
   chettu
   ```

2. Scan a specific directory:
   ```
   chettu -d /path/to/project
   ```

3. Scan multiple directories:
   ```
   chettu -d /path/to/project1 -d /path/to/project2
   ```

4. Add custom ignore patterns:
   ```
   chettu -l "*.log" -l "node_modules"
   ```

5. Use custom ignore files:
   ```
   chettu -f .customignore -f .projectignore
   ```

6. Reset default ignore rules and use only custom ones:
   ```
   chettu --reset-ignore -l "*.tmp" -f .customignore
   ```

7. Copy to clipboard with a 100KB limit:
   ```
   chettu -c 102400
   ```

8. Write output to a file:
   ```
   chettu -o project_structure.xml
   ```

9. Force overwrite existing output file:
   ```
   chettu -o project_structure.xml -R
   ```

10. Combine multiple options:
    ```
    chettu -d /path/to/project -l "*.log" -f .customignore -c 200000 -o output.xml
    ```

## Output

The program generates an XML output with the following structure:

```xml
<project>
  <file_tree>
    <file_path>path/to/file1</file_path>
    <file_path>path/to/file2</file_path>
    ...
  </file_tree>
  <file>
    <file_path>path/to/file1</file_path>
    <file_content>
      // File contents here
    </file_content>
  </file>
  ...
</project>
```

## Dependencies

- github.com/atotto/clipboard
- github.com/sabhiram/go-gitignore
- github.com/spf13/pflag
