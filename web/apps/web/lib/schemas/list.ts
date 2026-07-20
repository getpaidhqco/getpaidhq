import {z} from "zod"

export const ListResponseSchema = z.object({
  data: z.array(z.any()),
  meta: z.object({
    total: z.number(),
    page: z.number(),
    limit: z.number(),
  })
})
export type  ListResponse = z.infer<typeof ListResponseSchema>
