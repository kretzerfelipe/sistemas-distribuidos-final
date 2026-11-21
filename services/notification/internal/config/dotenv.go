package config

import (
	"bufio"
	"os"
	"strings"
)

func loadDotEnv(paths ...string) {
	for _, path := range paths {
		if loadDotEnvFile(path) {
			return
		}
	}
}

func loadDotEnvFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		setVariableFromLine(scanner.Text())
	}
	return true
}

func setVariableFromLine(rawLine string) {
	line := strings.TrimSpace(rawLine)
	lineIsBlankOrComment := line == "" || strings.HasPrefix(line, "#")
	if lineIsBlankOrComment {
		return
	}

	key, value, hasAssignment := strings.Cut(line, "=")
	if !hasAssignment {
		return
	}
	key = strings.TrimSpace(key)
	value = strings.Trim(strings.TrimSpace(value), `"'`)

	_, variableIsAlreadySet := os.LookupEnv(key)
	if !variableIsAlreadySet {
		_ = os.Setenv(key, value)
	}
}
