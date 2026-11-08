package pixellab

// Error is the message the API sent, or the last of the validation errors.
func (e *HTTPValidationError) Error() string {
	switch d := e.Detail; {
	case d.String != "":
		return d.String
	case len(d.ValidationError) > 0:
		return d.ValidationError[len(d.ValidationError)-1].Msg
	default:
		return "validation error"
	}
}
