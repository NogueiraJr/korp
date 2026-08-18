import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, catchError, map, of } from 'rxjs';
import { HealthStatus } from '../models/api.model';

@Injectable({ providedIn: 'root' })
export class HealthService {
  constructor(private readonly http: HttpClient) {}

  /** Polls a microservice health endpoint and turns failures into a friendly
   *  'down' status instead of throwing. */
  check(service: 'estoque' | 'faturamento'): Observable<HealthStatus> {
    const url = service === 'estoque' ? '/api/estoque/health' : '/api/faturamento/health';
    return this.http.get<HealthStatus>(url).pipe(
      map((h) => ({ ...h, status: 'up' })),
      catchError(() => of({ status: 'down', service })),
    );
  }
}