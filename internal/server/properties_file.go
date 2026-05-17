package server

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type propertyLine struct {
	raw   string
	key   string
	value string
	isKV  bool
}

type propertiesFile struct {
	lines []propertyLine
}

func parsePropertiesFile(path string) (*propertiesFile, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &propertiesFile{}, nil
		}
		return nil, err
	}
	defer file.Close()

	props := &propertiesFile{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		rawLine := scanner.Text()
		line := propertyLine{raw: rawLine}

		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			props.lines = append(props.lines, line)
			continue
		}

		parts := strings.SplitN(rawLine, "=", 2)
		if len(parts) != 2 {
			props.lines = append(props.lines, line)
			continue
		}

		key := strings.TrimSpace(parts[0])
		if key == "" {
			props.lines = append(props.lines, line)
			continue
		}

		line.key = key
		line.value = strings.TrimSpace(parts[1])
		line.isKV = true
		props.lines = append(props.lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return props, nil
}

func (p *propertiesFile) Get(key string) (string, bool) {
	var foundVal string
	found := false
	for _, line := range p.lines {
		if line.isKV && line.key == key {
			foundVal = line.value
			found = true
		}
	}
	return foundVal, found
}

func (p *propertiesFile) SetMany(updates map[string]string) {
	applied := make(map[string]bool, len(updates))

	for i := range p.lines {
		line := &p.lines[i]
		if !line.isKV {
			continue
		}
		newVal, ok := updates[line.key]
		if !ok {
			continue
		}

		line.value = newVal
		line.raw = fmt.Sprintf("%s=%s", line.key, newVal)
		applied[line.key] = true
	}

	for key, value := range updates {
		if applied[key] {
			continue
		}
		p.lines = append(p.lines, propertyLine{
			raw:   fmt.Sprintf("%s=%s", key, value),
			key:   key,
			value: value,
			isKV:  true,
		})
	}
}

func (p *propertiesFile) Write(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create properties directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range p.lines {
		if _, err := writer.WriteString(line.raw + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}
