import '@tanstack/react-table'

declare module '@tanstack/react-table' {
  interface TableMeta<TData = any> {
    refetch?: () => void
  }
}

export interface PaginationParams {
  page?: number
  limit?: number
}
