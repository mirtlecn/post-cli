package clipboard

type Service interface {
	ReadText() (string, error)
	WriteText(text string) error
}
