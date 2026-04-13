package quickstart

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
)

// downloadAndExtractSnapshot downloads a .tar.lz4 snapshot and extracts it to homeDir.
// Requires `curl`, `lz4`, and `tar` commands to be available in PATH.
func downloadAndExtractSnapshot(snapshotURL, homeDir string) error {
	// Validate URL to prevent command injection
	u, err := url.Parse(snapshotURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("invalid snapshot URL: %s (must be http or https)", snapshotURL)
	}

	for _, tool := range []string{"curl", "lz4", "tar"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s command not found; required for snapshot extraction", tool)
		}
	}

	// Use separate exec.Command calls to avoid shell injection
	curl := exec.Command("curl", "-o", "-", "-L", snapshotURL)
	lz4 := exec.Command("lz4", "-c", "-d", "-")
	tar := exec.Command("tar", "-x", "-C", homeDir)

	// Pipe: curl -> lz4 -> tar
	var err1, err2 error
	lz4.Stdin, err1 = curl.StdoutPipe()
	if err1 != nil {
		return fmt.Errorf("failed to create curl->lz4 pipe: %w", err1)
	}
	tar.Stdin, err2 = lz4.StdoutPipe()
	if err2 != nil {
		return fmt.Errorf("failed to create lz4->tar pipe: %w", err2)
	}
	tar.Stdout = os.Stdout
	tar.Stderr = os.Stderr

	// Start in reverse order
	if err := tar.Start(); err != nil {
		return fmt.Errorf("failed to start tar: %w", err)
	}
	if err := lz4.Start(); err != nil {
		return fmt.Errorf("failed to start lz4: %w", err)
	}
	if err := curl.Start(); err != nil {
		return fmt.Errorf("failed to start curl: %w", err)
	}

	// Wait for all processes
	if err := curl.Wait(); err != nil {
		return fmt.Errorf("curl failed: %w", err)
	}
	if err := lz4.Wait(); err != nil {
		return fmt.Errorf("lz4 failed: %w", err)
	}
	if err := tar.Wait(); err != nil {
		return fmt.Errorf("tar failed: %w", err)
	}

	return nil
}
