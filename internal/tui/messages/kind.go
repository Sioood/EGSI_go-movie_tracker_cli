package messages

// Kind classifies user-facing status messages for styling.
type Kind int

const (
	KindInfo Kind = iota
	KindSuccess
	KindError
)
