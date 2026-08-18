import { Injectable } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { HttpErrorResponse } from '@angular/common/http';

@Injectable({ providedIn: 'root' })
export class NotificationService {
  constructor(private readonly snackBar: MatSnackBar) {}

  success(message: string): void {
    this.snackBar.open(message, 'OK', { duration: 3500, panelClass: 'snack-success' });
  }

  error(message: string): void {
    this.snackBar.open(message, 'Fechar', { duration: 6000, panelClass: 'snack-error' });
  }

  fromHttpError(err: unknown, fallback: string): void {
    if (err instanceof HttpErrorResponse) {
      const body = err.error as { error?: string } | undefined;
      this.error(body?.error ?? fallback);
    } else {
      this.error(fallback);
    }
  }
}