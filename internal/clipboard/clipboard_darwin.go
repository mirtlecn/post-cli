//go:build darwin

package clipboard

var darwinReadCandidates = []commandCandidate{
	{name: "pbpaste"},
}

var darwinWriteCandidates = []commandCandidate{
	{name: "pbcopy"},
}

type SystemService struct{}

func NewSystemService() *SystemService {
	return &SystemService{}
}

func (service *SystemService) ReadText() (string, error) {
	command, err := resolveCommand(darwinReadCandidates, "pbpaste not found")
	if err != nil {
		return "", err
	}

	return readTextWithCommand(command)
}

func (service *SystemService) CanWriteText() bool {
	return canResolveCommand(darwinWriteCandidates)
}

func (service *SystemService) WriteText(text string) error {
	command, err := resolveCommand(darwinWriteCandidates, "pbcopy not found")
	if err != nil {
		return err
	}

	return writeTextWithCommand(command, text)
}
