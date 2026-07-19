package postgresgorm

// row_helpers.go houses small utilities shared across <entity>Row → domain
// mappers. Add helpers here as patterns emerge — keep them small and
// purpose-built. Examples that will land later in the plan:
//   - timePtr / ptrTime for nullable timestamps that are not modeled with
//     serializer:nulltime
//   - copyMap[K comparable, V any] for shallow-copying serialized JSON maps
//     when mutation safety matters
