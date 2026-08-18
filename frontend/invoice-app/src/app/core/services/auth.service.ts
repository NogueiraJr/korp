import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject, Observable, tap } from 'rxjs';
import { LoginResponse } from '../models/api.model';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly tokenKey = 'korp_token';
  private readonly userKey = 'korp_user';

  private readonly isAuthenticatedSubject = new BehaviorSubject<boolean>(this.hasToken());

  /** RxJS: BehaviorSubject exposes the authentication state reactively. */
  isAuthenticated$: Observable<boolean> = this.isAuthenticatedSubject.asObservable();

  username: string = localStorage.getItem(this.userKey) ?? '';

  constructor(private readonly http: HttpClient) {}

  login(username: string, password: string): Observable<LoginResponse> {
    return this.http
      .post<LoginResponse>('/api/faturamento/auth/login', { username, password })
      .pipe(
        tap((response) => {
          localStorage.setItem(this.tokenKey, response.token);
          localStorage.setItem(this.userKey, response.username);
          this.username = response.username;
          this.isAuthenticatedSubject.next(true);
        }),
      );
  }

  logout(): void {
    localStorage.removeItem(this.tokenKey);
    localStorage.removeItem(this.userKey);
    this.username = '';
    this.isAuthenticatedSubject.next(false);
  }

  get token(): string {
    return localStorage.getItem(this.tokenKey) ?? '';
  }

  private hasToken(): boolean {
    return this.token.length > 0;
  }
}