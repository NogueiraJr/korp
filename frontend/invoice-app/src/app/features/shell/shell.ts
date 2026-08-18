import { ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Router, RouterModule } from '@angular/router';
import { interval, Subject, takeUntil } from 'rxjs';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatTabsModule } from '@angular/material/tabs';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { CommonModule } from '@angular/common';
import { AuthService } from '../../core/services/auth.service';
import { HealthService } from '../../core/services/health.service';
import { ControlService } from '../../core/services/control.service';
import { HealthStatus, ServiceSource } from '../../core/models/api.model';

@Component({
  selector: 'app-shell',
  imports: [
    CommonModule,
    RouterModule,
    MatToolbarModule,
    MatTabsModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './shell.html',
  styleUrl: './shell.scss',
})
export class ShellComponent implements OnInit, OnDestroy {
  /** RxJS: Subject used with takeUntil to cancel subscriptions on destroy. */
  private readonly destroy$ = new Subject<void>();

  estoque: HealthStatus = { status: 'down', service: 'estoque' };
  faturamento: HealthStatus = { status: 'down', service: 'faturamento' };

  /** Service currently being started/stopped, or null when idle. */
  busy: ServiceSource | null = null;

  constructor(
    public readonly auth: AuthService,
    private readonly health: HealthService,
    private readonly control: ControlService,
    private readonly router: Router,
    private readonly cdr: ChangeDetectorRef,
  ) {}

  ngOnInit(): void {
    // Immediate check so the badges are correct right on load, then poll every
    // 5 seconds to keep them up to date.
    this.check('estoque');
    this.check('faturamento');

    interval(5000)
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => {
        this.check('estoque');
        this.check('faturamento');
      });
  }

  private check(source: ServiceSource): void {
    this.health
      .check(source)
      .pipe(takeUntil(this.destroy$))
      .subscribe((status) => {
        if (source === 'estoque') this.estoque = status;
        else this.faturamento = status;
        this.cdr.markForCheck();
      });
  }

  ngOnDestroy(): void {
    // Angular lifecycle: ngOnDestroy cancels all subscriptions to avoid leaks.
    this.destroy$.next();
    this.destroy$.complete();
  }

  isUp(source: ServiceSource): boolean {
    return (source === 'estoque' ? this.estoque : this.faturamento).status === 'up';
  }

  /** Starts a stopped service or stops a running one, then refreshes the badge. */
  toggleService(source: ServiceSource): void {
    if (this.busy) return;
    this.busy = source;
    this.cdr.markForCheck();

    const action = this.isUp(source) ? this.control.stop(source) : this.control.start(source);
    action.pipe(takeUntil(this.destroy$)).subscribe({
      next: () => this.afterToggle(source),
      error: () => this.afterToggle(source),
    });
  }

  private afterToggle(source: ServiceSource): void {
    this.busy = null;
    this.cdr.markForCheck();
    // Refresh the badge immediately instead of waiting for the next poll.
    this.health
      .check(source)
      .pipe(takeUntil(this.destroy$))
      .subscribe((status) => {
        if (source === 'estoque') this.estoque = status;
        else this.faturamento = status;
        this.cdr.markForCheck();
      });
  }

  logout(): void {
    this.auth.logout();
    this.router.navigate(['/login']);
  }
}