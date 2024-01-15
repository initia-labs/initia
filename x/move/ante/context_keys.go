package ante

// private type creates an interface key for Context that cannot be accessed by any other package
type contextKey int

const (
	GasPricesContextKey contextKey = iota
)
