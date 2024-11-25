package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

var unknownCounter atomic.Int32

// isFilePath checks if the line starts with a file comment
func isFilePath(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "#") {
		filename := strings.TrimSpace(strings.TrimPrefix(line, "#"))
		// Skip empty filenames and lines that are just comments
		if filename != "" && filename != "." && !strings.HasPrefix(filename, " ") {
			return filename, true
		}
	}
	return "", false
}

func ensureDirectoryExists(filepath string) error {
	dir := path.Dir(filepath)
	if dir != "." {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

func writeToFile(filename string, content []byte) error {
	if err := ensureDirectoryExists(filename); err != nil {
		return fmt.Errorf("error creating directory for %s: %w", filename, err)
	}

	if err := os.WriteFile(filename, content, 0644); err != nil {
		return fmt.Errorf("error writing file %s: %w", filename, err)
	}

	fmt.Printf("Created: %s\n", filename)
	return nil
}

func processYAML(input io.Reader) error {
	var currentContent strings.Builder
	var inDocument bool
	var firstLine string
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		line := scanner.Text()

		// Check for document separator
		if line == "---" {
			// Process previous document if exists
			if inDocument {
				if err := processDocument(firstLine, currentContent.String()); err != nil {
					return err
				}
			}
			// Reset for new document
			currentContent.Reset()
			inDocument = true
			firstLine = ""
			continue
		}

		// Store first non-empty line after separator
		if inDocument && firstLine == "" && strings.TrimSpace(line) != "" {
			firstLine = line
		}

		// Accumulate content
		if inDocument {
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}

	// Process last document
	if inDocument {
		return processDocument(firstLine, currentContent.String())
	}

	return scanner.Err()
}

func processDocument(firstLine, content string) error {
	// First try to get filepath from comment
	if filepath, ok := isFilePath(firstLine); ok {
		return writeToFile(filepath, []byte(content))
	}

	// Try to parse as YAML
	var doc map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		// If parsing fails, write to unknown file
		index := unknownCounter.Add(1)
		return writeToFile(fmt.Sprintf("unknown-%d.txt", index), []byte(content))
	}

	// Check for kind/metadata/name structure
	kind, _ := doc["kind"].(string)
	metadata, _ := doc["metadata"].(map[string]interface{})
	name, _ := metadata["name"].(string)

	if kind != "" && name != "" {
		filename := fmt.Sprintf("%s-%s.yaml", strings.ToLower(kind), name)
		return writeToFile(filename, []byte(content))
	}

	// If no valid filename could be constructed, use unknown
	index := unknownCounter.Add(1)
	return writeToFile(fmt.Sprintf("unknown-%d.txt", index), []byte(content))
}

func main() {
	flag.Parse()

	var err error
	if flag.NArg() == 0 {
		// Read from stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			fmt.Fprintln(os.Stderr, "Usage: yamlsplitter [filename] or cat yourfile.yaml | yamlsplitter")
			os.Exit(1)
		}
		err = processYAML(os.Stdin)
	} else {
		// Read from file
		filename := flag.Arg(0)
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file %s: %v\n", filename, err)
			os.Exit(1)
		}
		defer file.Close()
		err = processYAML(file)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
