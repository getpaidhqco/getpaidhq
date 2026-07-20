import {z} from "zod"

export const RecurringRevenueSchema = z.object({
  period: z.string(),
  total: z.number(),
  count: z.number().optional().nullable(),
  growth_mom: z.number().optional().nullable(),
  type: z.string(),
})
export type RecurringRevenue = z.infer<typeof RecurringRevenueSchema>


