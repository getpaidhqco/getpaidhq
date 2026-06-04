package service

// Dunning command input types are defined in internal/core/port/dunning_input.go
// because they appear in port.DunningService and port.DunningEngine interface
// signatures. They are re-exported here as type aliases for service-layer
// convenience so callers can use service.* or port.* interchangeably.
//
// (No types defined here — see port/dunning_input.go.)
