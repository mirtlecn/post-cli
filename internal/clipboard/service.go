package clipboard

type Service interface {
	ReadText() (string, error)
	CanWriteText() bool
	WriteText(text string) error
}
