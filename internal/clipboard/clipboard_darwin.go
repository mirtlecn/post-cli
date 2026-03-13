//go:build darwin

package clipboard

import (
	"fmt"
	"sync"

	systemclipboard "golang.design/x/clipboard"
)

type SystemService struct{}

var initializeOnce sync.Once
var initializeError error

func NewSystemService() *SystemService {
	return &SystemService{}
}

func (service *SystemService) ReadText() (string, error) {
	if err := initializeClipboard(); err != nil {
		return "", err
	}

	content := systemclipboard.Read(systemclipboard.FmtText)
	if len(content) == 0 {
		return "", fmt.Errorf("clipboard is empty")
	}
	return string(content), nil
}

func (service *SystemService) WriteText(text string) error {
	if err := initializeClipboard(); err != nil {
		return err
	}

	systemclipboard.Write(systemclipboard.FmtText, []byte(text))
	return nil
}

func initializeClipboard() error {
	initializeOnce.Do(func() {
		initializeError = systemclipboard.Init()
	})
	if initializeError != nil {
		return fmt.Errorf("clipboard unavailable: %w", initializeError)
	}
	return nil
}
