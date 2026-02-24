package cluster

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func setTOMLValue(path, section, key, value string) error {
	bz, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(bz), "\n")
	sectionHeader := "[" + section + "]"
	inSection := false
	updated := false
	keyRe := regexp.MustCompile(`^` + regexp.QuoteMeta(key) + `\s*=`)

	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inSection = line == sectionHeader
			continue
		}
		if !inSection {
			continue
		}
		if keyRe.MatchString(line) {
			indent := raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))]
			lines[i] = fmt.Sprintf("%s%s = %s", indent, key, value)
			updated = true
			break
		}
	}

	if !updated {
		return fmt.Errorf("failed to set %s.%s in %s", section, key, path)
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}
