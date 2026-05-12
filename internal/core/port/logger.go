package port

import "getpaidhq/internal/lib"

// Logger is an alias for lib.Logger - the core logging interface.
// Defined in lib to avoid import cycles (domain -> lib -> port -> domain).
type Logger = lib.Logger
