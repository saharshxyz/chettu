# Chettu (చెట్టు) - File Tree Generator

Chettu is a Go script that generates a structured output of file trees and their contents, with options for ignoring specific files/directories and controlling the output format.

## Features

- Traverse directories and generate a file tree
- Include file contents in the output
- Ignore files/directories based on patterns (similar to `.gitignore`)
- Copy output to clipboard
- Write output to a file
- Customizable ignore patterns and files

## Usage

```
go run chettu.go [flags]
```

## Flags

### `-d <directory>`
Specifies directories to process. Can be used multiple times to process multiple directories.
- Default: Current directory (`"."`)
- Example: `-d ./src -d ./tests`
- Note: The script uses absolute file paths internally but outputs relative paths.

### `-i <ignore_pattern>`
Specifies paths to ignore. Can be used multiple times.
- Use `-i ""` to clear default ignore patterns.
- Example: `-i "*.log" -i "tmp/"`
- Note: If no ignore patterns or files are specified, it will attempt to use .gitignore in the current directory.

### `-if <file_path>`
Specifies files containing ignore patterns (similar to .gitignore). Can be used multiple times.
- Default: `".gitignore"`
- Use `-if ""` to prevent loading any ignore files.
- Example: `-if .customignore`
- Note: If no ignore patterns or files are specified, it will attempt to use .gitignore in the current directory.

### `-c [max_size]`
Enables clipboard copy with an optional maximum size.
- Default max size: `500000` characters
- Use `-c` without a value to use the default max size
- Example with custom size: `-c 1000000`
- Note: The clipboard feature may not work on all operating systems. Ensure you have the necessary dependencies installed.

### `-of <file_path>`
Specifies the output file path. If not provided, output is printed to stdout (unless `-c` is used).
- Example: `-of output.xml`
- Note: When using this flag without `-ofr`, the script will prompt before overwriting an existing file.

### `-ofr`
Forces replacement of an existing output file without prompting.
- Must be used with `-of`.
- Example: `-of output.xml -ofr`

## Output Format

Chettu generates output in an XML-like format:

```xml
<documents>
<file_path1>
<file_path2>
    ...
    <document>
        <source>file_path1</source>
        <document_content>
            [File contents here]
        </document_content>
    </document>
    <document>
        <source>file_path2</source>
        <document_content>
            [File contents here]
        </document_content>
    </document>
    ...
</documents>
```

## Examples

1. Process current directory, use default .gitignore, and print to stdout:
   ```
   go run chettu.go
   ```

2. Process multiple directories, ignore specific patterns, and copy to clipboard:
   ```
   go run chettu.go -d ./src -d ./tests -i "*.log" -i "tmp/" -c=
   ```

3. Use a custom ignore file and write output to a file:
   ```
   go run chettu.go -if .customignore -of output.xml
   ```

4. Process a specific directory, clear default ignores, and force replace output file:
   ```
   go run chettu.go -d ./project -i "" -of output.xml -ofr
   ```