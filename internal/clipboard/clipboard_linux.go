//go:build linux

package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
)

type SystemService struct{}

func NewSystemService() *SystemService {
	return &SystemService{}
}

func (service *SystemService) ReadText() (string, error) {
	command, args, err := readClipboardCommand()
	if err != nil {
		return "", err
	}

	output, runErr := exec.Command(command, args...).Output()
	if runErr != nil {
		return "", fmt.Errorf("clipboard unavailable: %w", runErr)
	}
	if len(output) == 0 {
		return "", fmt.Errorf("clipboard is empty")
	}
	return string(output), nil
}

func (service *SystemService) WriteText(text string) error {
	command, args, err := writeClipboardCommand()
	if err != nil {
		return err
	}

	process := exec.Command(command, args...)
	process.Stdin = bytes.NewBufferString(text)
	if output, runErr := process.CombinedOutput(); runErr != nil {
		if len(output) > 0 {
			return fmt.Errorf("clipboard unavailable: %s", string(output))
		}
		return fmt.Errorf("clipboard unavailable: %w", runErr)
	}
	return nil
}

func readClipboardCommand() (string, []string, error) {
	if path, err := exec.LookPath("wl-paste"); err == nil {
		return path, []string{"--no-newline"}, nil
	}
	if path, err := exec.LookPath("xclip"); err == nil {
		return path, []string{"-selection", "clipboard", "-o"}, nil
	}
	if path, err := exec.LookPath("xsel"); err == nil {
		return path, []string{"--clipboard", "--output"}, nil
	}
	return "", nil, fmt.Errorf("clipboard unavailable: no clipboard read command found")
}

func writeClipboardCommand() (string, []string, error) {
	if path, err := exec.LookPath("wl-copy"); err == nil {
		return path, nil, nil
	}
	if path, err := exec.LookPath("xclip"); err == nil {
		return path, []string{"-selection", "clipboard"}, nil
	}
	if path, err := exec.LookPath("xsel"); err == nil {
		return path, []string{"--clipboard", "--input"}, nil
	}
	return "", nil, fmt.Errorf("clipboard unavailable: no clipboard write command found")
}
