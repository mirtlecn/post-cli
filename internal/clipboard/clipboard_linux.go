//go:build linux

package clipboard

var linuxReadCandidates = []commandCandidate{
	{name: "wl-paste", args: []string{"--no-newline"}},
	{name: "xclip", args: []string{"-selection", "clipboard", "-o"}},
	{name: "xsel", args: []string{"--clipboard", "--output"}},
}

var linuxWriteCandidates = []commandCandidate{
	{name: "wl-copy"},
	{name: "xclip", args: []string{"-selection", "clipboard"}},
	{name: "xsel", args: []string{"--clipboard", "--input"}},
}

type SystemService struct{}

func NewSystemService() *SystemService {
	return &SystemService{}
}

func (service *SystemService) ReadText() (string, error) {
	command, err := resolveCommand(linuxReadCandidates, "no clipboard read command found")
	if err != nil {
		return "", err
	}

	return readTextWithCommand(command)
}

func (service *SystemService) CanWriteText() bool {
	return canResolveCommand(linuxWriteCandidates)
}

func (service *SystemService) WriteText(text string) error {
	command, err := resolveCommand(linuxWriteCandidates, "no clipboard write command found")
	if err != nil {
		return err
	}

	return writeTextWithCommand(command, text)
}
