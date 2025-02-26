package common

import (
	"bufio"
	"os"
	"strings"
)

func LoadRuntimeConfig() map[string]string {
	d := make(map[string]string)
	f, err := os.Open(".exorc")
	if err != nil {
		return d
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue
		}

		parts := strings.Split(line, "->")
		if len(parts) != 2 {
			continue
		}

		d[parts[0]] = parts[1]
	}

	return d
}
