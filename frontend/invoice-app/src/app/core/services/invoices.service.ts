import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';
import { Invoice, InvoicePayload } from '../models/invoice.model';

const BASE = '/api/faturamento';

@Injectable({ providedIn: 'root' })
export class InvoicesService {
  constructor(private readonly http: HttpClient) {}

  list(): Observable<Invoice[]> {
    return this.http.get<Invoice[]>(`${BASE}/invoices`);
  }

  create(payload: InvoicePayload): Observable<Invoice> {
    return this.http.post<Invoice>(`${BASE}/invoices`, payload);
  }

  /** Prints (closes) an invoice. The Idempotency-Key header guarantees that
   *  repeating the same request has no side effects. */
  print(id: string, idempotencyKey: string): Observable<Invoice> {
    const headers = new HttpHeaders({ 'Idempotency-Key': idempotencyKey });
    return this.http.post<Invoice>(`${BASE}/invoices/${id}/print`, null, { headers });
  }
}