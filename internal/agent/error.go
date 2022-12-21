package agent

type DecryptError struct {
	msg string
}

func (de *DecryptError) Error() string {
	return de.msg
}

func NewDecryptError(text string) error {
	return &DecryptError{msg: text}
}
