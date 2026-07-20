import { useCustomers, useOrders, usePayments } from '@getpaidhq/react-sdk'
import { ResourceType } from '@/components/atoms/datatable/ResourceDataTable'

interface UseResourceDataParams {
  resourceType: ResourceType
  params: {
    page: number
    limit: number
  }
  initialData?: any[]
}

export function useResourceData({ resourceType, params, initialData = [] }: UseResourceDataParams) {
  // Use the appropriate hook based on resourceType
  switch (resourceType) {
    case 'customers':
      return useCustomers(params)
    case 'orders':
      return useOrders(params)
    case 'payments':
      return usePayments(params)
    default:
      // Return a default query result structure for initialData
      return {
        data: {
          data: initialData,
          meta: {
            total: initialData.length
          }
        },
        isLoading: false
      }
  }
}
