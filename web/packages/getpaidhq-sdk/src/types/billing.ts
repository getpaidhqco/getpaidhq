/** Invoice reminder configuration (spec: ReminderConfigDTO). */
export interface ReminderConfigDTO {
  enabled?: boolean;
  /** Reminder offsets, e.g. ["-3d", "0d", "7d"]. */
  offsets: string[];
}
