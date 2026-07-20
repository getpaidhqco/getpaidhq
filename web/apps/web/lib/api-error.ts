import { GetPaidHQError } from "@getpaidhq/sdk"

/**
 * The server returns field-level validation failures as a `details`/`errors` array
 * of `{ more, name, reason }` entries (see HTTPErrorDetail). `more` carries the
 * go-playground/validator fields: `field`, `tag`, `param`, `value`.
 */
interface ValidatorMore {
  field?: string
  tag?: string
  param?: string
  value?: unknown
}

interface ServerErrorDetail {
  more?: Record<string, unknown>
  name?: string
  reason?: string
}

/** A human-readable line for a single server validation detail. */
function describeDetail(detail: ServerErrorDetail): string {
  const more = (detail.more ?? {}) as ValidatorMore
  const field = more.field ?? "Field"
  const options = String(more.param ?? "")
    .split(" ")
    .filter(Boolean)
    .join(", ")

  switch (more.tag) {
    case "oneof":
      return `${field}: must be one of ${options}`
    case "required":
      return `${field} is required`
    case "min":
      return `${field}: must be at least ${more.param}`
    case "max":
      return `${field}: must be at most ${more.param}`
    case "gte":
      return `${field}: must be ≥ ${more.param}`
    case "lte":
      return `${field}: must be ≤ ${more.param}`
    default:
      return detail.reason || `${field}: invalid value`
  }
}

/**
 * Turn any thrown error into a toast-friendly `{ message, description }`. For a
 * server validation error this expands the field-level `details` into readable
 * lines instead of the bare "Validation Error" title.
 */
export function formatApiError(error: unknown): { message: string; description?: string } {
  if (error instanceof GetPaidHQError) {
    const details = error.details
    if (Array.isArray(details) && details.length > 0) {
      return {
        message: error.message || "Validation error",
        description: (details as ServerErrorDetail[]).map(describeDetail).join("\n"),
      }
    }
    return { message: error.message }
  }
  if (error instanceof Error) return { message: error.message }
  return { message: "Something went wrong" }
}
