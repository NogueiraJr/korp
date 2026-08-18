import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ActivityLog } from '../models/api.model';

@Injectable({ providedIn: 'root' })
export class LogsService {
  constructor(private readonly http: HttpClient) {}

  /** Fetches the recent activity logs from a specific microservice. */
  fetch(service: 'estoque' | 'faturamento', limit = 200): Observable<ActivityLog[]> {
    const base = service === 'estoque' ? '/api/estoque' : '/api/faturamento';
    return this.http.get<ActivityLog[]>(`${base}/logs`, { params: { limit: String(limit) } });
  }
}