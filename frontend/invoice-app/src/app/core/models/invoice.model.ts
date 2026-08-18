export type InvoiceStatus = 'OPEN' | 'CLOSED';

export interface InvoiceItem {
  id?: string;
  product_code: string;
  quantity: number;
}

export interface InvoiceItemPayload {
  product_code: string;
  quantity: number;
}

export interface Invoice {
  id: string;
  number: number;
  status: InvoiceStatus;
  items: InvoiceItem[];
  created_at: string;
  closed_at?: string | null;
}

export interface InvoicePayload {
  items: InvoiceItemPayload[];
}