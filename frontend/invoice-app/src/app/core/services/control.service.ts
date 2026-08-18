import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ControlStatus, ServiceSource } from '../models/api.model';

@Injectable({ providedIn: 'root' })
export class ControlService {
  constructor(private readonly http: HttpClient) {}

  status(): Observable<ControlStatus> {
    return this.http.get<ControlStatus>('/api/control/status');
  }

  start(source: ServiceSource): Observable<{ ok: boolean; running: boolean }> {
    return this.http.post<{ ok: boolean; running: boolean }>(`/api/control/${source}/start`, {});
  }

  stop(source: ServiceSource): Observable<{ ok: boolean; running: boolean }> {
    return this.http.post<{ ok: boolean; running: boolean }>(`/api/control/${source}/stop`, {});
  }
}