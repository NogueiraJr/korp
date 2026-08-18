import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { Product, ProductPayload } from '../models/product.model';

const BASE = '/api/estoque';

@Injectable({ providedIn: 'root' })
export class ProductsService {
  constructor(private readonly http: HttpClient) {}

  list(): Observable<Product[]> {
    return this.http.get<Product[]>(`${BASE}/products`);
  }

  create(payload: ProductPayload): Observable<Product> {
    return this.http.post<Product>(`${BASE}/products`, payload);
  }

  update(code: string, payload: { description: string; stock_quantity: number }): Observable<Product> {
    return this.http.put<Product>(`${BASE}/products/${code}`, payload);
  }

  suggestDescription(code: string): Observable<{ suggestion: string }> {
    return this.http.post<{ suggestion: string }>(`${BASE}/products/ai-description`, { code });
  }
}