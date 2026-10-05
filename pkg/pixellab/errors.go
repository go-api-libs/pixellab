package pixellab

func (e *SimpleError) Error() string {
	return e.Detail
}

// TODO
func (e *HTTPValidationError) Error() string {
	return e.Detail[len(e.Detail)-1].Msg
}
