import type {ProductResponse} from "@getpaidhq/sdk";
import {AuthHeader} from "@getpaidhq/auth";


export async function fetchData(id: string, authHeaders: AuthHeader): Promise<ProductResponse> {
  const rsp = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/products/${id}`, {
    headers: authHeaders
  })
    .then((res) => res.json());

  return rsp as ProductResponse
}
