package clipboard

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type commandSpec struct {
	path string
	args []string
}

type commandCandidate struct {
	name string
	args []string
}

func readTextWithCommand(command commandSpec) (string, error) {
	output, runError := exec.Command(command.path, command.args...).Output()
	if runError != nil {
		return "", fmt.Errorf("clipboard unavailable: %w", runError)
	}

	content := strings.TrimRight(string(output), "\r\n")
	if content == "" {
		return "", fmt.Errorf("clipboard is empty")
	}
	return content, nil
}

func writeTextWithCommand(command commandSpec, text string) error {
	process := exec.Command(command.path, command.args...)
	process.Stdin = bytes.NewBufferString(text)

	output, runError := process.CombinedOutput()
	if runError != nil {
		if len(output) > 0 {
			return fmt.Errorf("clipboard unavailable: %s", strings.TrimSpace(string(output)))
		}
		return fmt.Errorf("clipboard unavailable: %w", runError)
	}
	return nil
}

func resolveCommand(candidates []commandCandidate, missingMessage string) (commandSpec, error) {
	for _, candidate := range candidates {
		path, lookupError := exec.LookPath(candidate.name)
		if lookupError == nil {
			return commandSpec{path: path, args: candidate.args}, nil
		}
	}

	return commandSpec{}, fmt.Errorf("clipboard unavailable: %s", missingMessage)
}

func canResolveCommand(candidates []commandCandidate) bool {
	for _, candidate := range candidates {
		if _, lookupError := exec.LookPath(candidate.name); lookupError == nil {
			return true
		}
	}

	return false
}
