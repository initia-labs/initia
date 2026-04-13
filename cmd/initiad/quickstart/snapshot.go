package quickstart

import (
	"fmt"
	"os"
	"os/exec"
)

// downloadAndExtractSnapshot downloads a .tar.lz4 snapshot and extracts it to homeDir.
// Requires `lz4` and `tar` commands to be available in PATH.
func downloadAndExtractSnapshot(url, homeDir string) error {
	if _, err := exec.LookPath("lz4"); err != nil {
		return fmt.Errorf("lz4 command not found. Install it with: apt install lz4 (Linux) or brew install lz4 (macOS)")
	}

	// Stream download -> lz4 decompress -> tar extract
	// Equivalent to: curl -o - -L <url> | lz4 -c -d - | tar -x -C <homeDir>
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf(`curl -o - -L "%s" | lz4 -c -d - | tar -x -C "%s"`, url, homeDir))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download and extract snapshot: %w", err)
	}

	return nil
}
