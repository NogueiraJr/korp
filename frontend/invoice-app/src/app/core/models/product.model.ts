export interface Product {
  code: string;
  description: string;
  stock_quantity: number;
  created_at?: string;
  updated_at?: string;
}

export interface ProductPayload {
  code: string;
  description: string;
  stock_quantity: number;
}