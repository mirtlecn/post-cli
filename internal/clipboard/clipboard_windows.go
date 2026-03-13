//go:build windows

package clipboard

var windowsClipboardCommands = []commandCandidate{
	{name: "powershell.exe", args: []string{"-NoProfile", "-NonInteractive", "-Command"}},
	{name: "powershell", args: []string{"-NoProfile", "-NonInteractive", "-Command"}},
	{name: "pwsh.exe", args: []string{"-NoProfile", "-NonInteractive", "-Command"}},
	{name: "pwsh", args: []string{"-NoProfile", "-NonInteractive", "-Command"}},
}

type SystemService struct{}

func NewSystemService() *SystemService {
	return &SystemService{}
}

func (service *SystemService) ReadText() (string, error) {
	command, err := resolveCommand(windowsReadCandidates(), "no PowerShell clipboard command found")
	if err != nil {
		return "", err
	}

	return readTextWithCommand(command)
}

func (service *SystemService) CanWriteText() bool {
	return canResolveCommand(windowsWriteCandidates())
}

func (service *SystemService) WriteText(text string) error {
	command, err := resolveCommand(windowsWriteCandidates(), "no PowerShell clipboard command found")
	if err != nil {
		return err
	}

	return writeTextWithCommand(command, text)
}

func windowsReadCandidates() []commandCandidate {
	candidates := make([]commandCandidate, 0, len(windowsClipboardCommands))
	for _, candidate := range windowsClipboardCommands {
		candidates = append(candidates, commandCandidate{
			name: candidate.name,
			args: append(append([]string{}, candidate.args...), "Get-Clipboard -Raw"),
		})
	}
	return candidates
}

func windowsWriteCandidates() []commandCandidate {
	candidates := make([]commandCandidate, 0, len(windowsClipboardCommands))
	for _, candidate := range windowsClipboardCommands {
		candidates = append(candidates, commandCandidate{
			name: candidate.name,
			args: append(append([]string{}, candidate.args...), "Set-Clipboard -Value ([Console]::In.ReadToEnd())"),
		})
	}
	return candidates
}
