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

func isFilePath(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "#") {
		filename := strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if filename == "" || filename == "." || strings.HasPrefix(filename, " ") {
			return "", false
		}

		if strings.Contains(filename, "├") || strings.Contains(filename, "└") ||
			strings.Contains(filename, "--") || strings.Contains(filename, "│") {
			return "", false
		}

		if strings.Contains(filename, "/") {
			return filename, true
		}

		if strings.Contains(filename, ".") || filename == "Makefile" {
			if strings.Contains(filename, "/") {
				return filename, true
			}
			return strings.TrimSpace(filename), true
		}
	}
	return "", false
}

func isLikelyContent(s string) bool {
	if strings.Contains(s, "├──") || strings.Contains(s, "└──") {
		return false
	}

	indicators := []string{
		"apiVersion:",
		"kind:",
		"metadata:",
		"spec:",
		"data:",
		"rules:",
		".PHONY",
		"#!/",
	}

	for _, indicator := range indicators {
		if strings.Contains(s, indicator) {
			return true
		}
	}
	return false
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
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024*10) // 10MB buffer

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if line == "---" {
			if inDocument && currentContent.Len() > 0 {
				if err := processDocument(firstLine, currentContent.String()); err != nil {
					return err
				}
			}
			currentContent.Reset()
			inDocument = true
			firstLine = ""
			continue
		}

		if !inDocument && trimmedLine != "" && strings.HasPrefix(trimmedLine, "#") {
			if _, ok := isFilePath(line); ok {
				if currentContent.Len() > 0 && isLikelyContent(currentContent.String()) {
					if err := processDocument(firstLine, currentContent.String()); err != nil {
						return err
					}
					currentContent.Reset()
				}
				firstLine = line
				inDocument = true
				continue
			}
		}

		if inDocument && firstLine == "" && trimmedLine != "" {
			firstLine = line
		}

		currentContent.WriteString(line)
		currentContent.WriteString("\n")
	}

	if (inDocument || currentContent.Len() > 0) && isLikelyContent(currentContent.String()) {
		return processDocument(firstLine, currentContent.String())
	}

	return scanner.Err()
}

func processDocument(firstLine, content string) error {
	if filepath, ok := isFilePath(firstLine); ok && strings.TrimSpace(content) != "" {
		if err := os.MkdirAll(path.Dir(filepath), 0755); err != nil {
			return fmt.Errorf("error creating directories for %s: %w", filepath, err)
		}
		return writeToFile(filepath, []byte(strings.TrimSpace(content)+"\n"))
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		index := unknownCounter.Add(1)
		return writeToFile(fmt.Sprintf("unknown-%d.txt", index), []byte(content))
	}

	kind, _ := doc["kind"].(string)
	metadata, _ := doc["metadata"].(map[string]interface{})
	name, _ := metadata["name"].(string)

	if kind != "" && name != "" {
		filename := fmt.Sprintf("%s-%s.yaml", strings.ToLower(kind), name)
		return writeToFile(filename, []byte(content))
	}

	index := unknownCounter.Add(1)
	return writeToFile(fmt.Sprintf("unknown-%d.txt", index), []byte(content))
}

func main() {
	flag.Parse()

	var err error
	if flag.NArg() == 0 {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			fmt.Fprintln(os.Stderr, "Usage: yamlsplitter [filename] or cat yourfile.yaml | yamlsplitter")
			os.Exit(1)
		}
		err = processYAML(os.Stdin)
	} else {
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
