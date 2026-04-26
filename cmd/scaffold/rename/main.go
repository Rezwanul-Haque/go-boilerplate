package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: make rename name=<new-project-name>")
		os.Exit(1)
	}

	newName := strings.TrimSpace(os.Args[1])
	if newName == "" {
		fmt.Fprintln(os.Stderr, "project name cannot be empty")
		os.Exit(1)
	}

	oldName, err := readModuleName("go.mod")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading go.mod: %v\n", err)
		os.Exit(1)
	}

	if oldName == newName {
		fmt.Printf("already named %q — nothing to do.\n", newName)
		return
	}

	// e.g. "go-boilerplate" → "Go Boilerplate", "my-api" → "My Api"
	oldTitle := toTitle(oldName)
	newTitle := toTitle(newName)

	fmt.Printf("renaming %q → %q\n", oldName, newName)
	fmt.Printf("title:   %q → %q\n\n", oldTitle, newTitle)

	changed := 0
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && shouldSkipDir(info.Name()) {
			return filepath.SkipDir
		}
		if !shouldProcess(path) {
			return nil
		}
		if replaced, rerr := replaceInFile(path, oldName, newName, oldTitle, newTitle); rerr != nil {
			fmt.Fprintf(os.Stderr, "  warning: %s: %v\n", path, rerr)
		} else if replaced {
			fmt.Printf("  updated  %s\n", path)
			changed++
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "walk error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Done. %d file(s) updated.\n", changed)
	fmt.Println("  run 'make swagger' to regenerate swagger docs.")
}

func readModuleName(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}
	return "", fmt.Errorf("module declaration not found in %s", path)
}

func replaceInFile(path, oldName, newName, oldTitle, newTitle string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	// replace title first (more specific), then module name
	updated := bytes.ReplaceAll(data, []byte(oldTitle), []byte(newTitle))
	updated = bytes.ReplaceAll(updated, []byte(oldName), []byte(newName))
	if bytes.Equal(data, updated) {
		return false, nil
	}
	return true, os.WriteFile(path, updated, 0644)
}

// toTitle converts "go-boilerplate" → "Go Boilerplate"
func toTitle(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, p := range parts {
		if len(p) == 0 {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "vendor", "bin", "node_modules":
		return true
	}
	return false
}

func shouldProcess(path string) bool {
	switch filepath.Ext(path) {
	case ".go", ".mod", ".sum", ".yml", ".yaml", ".env", ".md":
		return true
	}
	return false
}
