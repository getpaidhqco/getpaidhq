/** Add cart item input (spec: AddItemRequest). */
export interface AddItemRequest {
  price_id: string;
  product_id: string;
  quantity?: number;
}

/** Remove cart item input (spec: RemoveItemRequest). */
export interface RemoveItemRequest {
  id?: string;
  org_id?: string;
}

/** Cart (spec: CartResponse). */
export interface CartResponse {
  data: any;
}
