package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	_ "path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func processYAML(input io.Reader) error {
	decoder := yaml.NewDecoder(input)

	for {
		// Create a map to store the YAML document
		var doc map[string]interface{}

		// Decode the next document
		err := decoder.Decode(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error decoding YAML: %w", err)
		}

		// Skip empty documents
		if doc == nil {
			continue
		}

		// Extract kind and name
		kind, _ := doc["kind"].(string)
		metadata, _ := doc["metadata"].(map[string]interface{})
		name, _ := metadata["name"].(string)

		if kind == "" || name == "" {
			continue
		}

		// Create output filename
		outfileName := fmt.Sprintf("%s-%s.yaml", strings.ToLower(kind), name)

		// Create output file
		outfile, err := os.Create(outfileName)
		if err != nil {
			return fmt.Errorf("error creating file %s: %w", outfileName, err)
		}
		defer outfile.Close()

		// Create encoder for output file
		encoder := yaml.NewEncoder(outfile)
		defer encoder.Close()

		// Write the document
		if err := encoder.Encode(doc); err != nil {
			return fmt.Errorf("error writing to file %s: %w", outfileName, err)
		}
	}

	return nil
}

func main() {
	// Parse command line flags
	flag.Parse()

	var err error
	if flag.NArg() == 0 {
		// No arguments - read from stdin
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
