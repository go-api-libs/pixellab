package pixellab

// Error is the message the API sent, or the last of the validation errors.
func (e *HTTPValidationError) Error() string {
	switch d := e.Detail; {
	case d.String != "":
		return d.String
	case len(d.ValidationErrors) > 0:
		return d.ValidationErrors[len(d.ValidationErrors)-1].Msg
	default:
		return "validation error"
	}
}
