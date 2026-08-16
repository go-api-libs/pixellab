package pixellab

// TODO
func (e *HTTPValidationError) Error() string {
	return e.Detail[len(e.Detail)-1].Msg
}
