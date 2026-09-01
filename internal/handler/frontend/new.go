package frontend

// New creates the SPA frontend handler.
func New() *Handler {
	return newHandler(embeddedFiles(), []byte(fallbackHTML))
}
