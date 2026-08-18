import { ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { interval, Subject, switchMap, takeUntil } from 'rxjs';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatChipsModule } from '@angular/material/chips';
import { MatTooltipModule } from '@angular/material/tooltip';
import { CommonModule } from '@angular/common';
import { LogsService } from '../../core/services/logs.service';
import { NotificationService } from '../../core/services/notification.service';
import { ActivityLog } from '../../core/models/api.model';
import { buildSummary, eventLabel, isErrorEvent } from '../../core/utils/activity-log.util';

interface DisplayLog extends ActivityLog {
  source: 'estoque' | 'faturamento';
}

@Component({
  selector: 'app-logs',
  imports: [
    CommonModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatProgressBarModule,
    MatChipsModule,
    MatTooltipModule,
  ],
  templateUrl: './logs.html',
  styleUrl: './logs.scss',
})
export class LogsComponent implements OnInit, OnDestroy {
  /** RxJS: Subject that triggers a reload; combined with switchMap it cancels
   *  any in-flight request when a new one is issued. */
  private readonly reload$ = new Subject<void>();
  private readonly destroy$ = new Subject<void>();

  logs: DisplayLog[] = [];
  loading = false;
  error: string | null = null;
  displayedColumns = ['created_at', 'source', 'event', 'summary'];

  constructor(
    private readonly logsService: LogsService,
    private readonly notify: NotificationService,
    private readonly cdr: ChangeDetectorRef,
  ) {}

  ngOnInit(): void {
    // Angular lifecycle: ngOnInit starts the polling + initial load.
    this.reload$
      .pipe(
        switchMap(() => this.load()),
        takeUntil(this.destroy$),
      )
      .subscribe();

    // RxJS: interval polls the log endpoints every 3 seconds while the tab is open.
    interval(3000)
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => this.reload$.next());

    this.reload$.next();
  }

  ngOnDestroy(): void {
    // Angular lifecycle: ngOnDestroy stops polling and cancels subscriptions.
    this.destroy$.next();
    this.destroy$.complete();
  }

  refresh(): void {
    this.reload$.next();
  }

  /** RxJS: forkJoin fetches the logs of both microservices in parallel,
   *  merges and sorts them newest-first. */
  private load() {
    this.loading = true;
    return this.logsService.fetch('estoque').pipe(
      switchMap((estoqueLogs) =>
        this.logsService.fetch('faturamento').pipe(
          switchMap((faturamentoLogs) => {
            const merged: DisplayLog[] = [
              ...estoqueLogs.map<DisplayLog>((l) => ({ ...l, source: 'estoque' })),
              ...faturamentoLogs.map<DisplayLog>((l) => ({ ...l, source: 'faturamento' })),
            ];
            merged.sort((a, b) => {
              const t = new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
              return t !== 0 ? t : b.id - a.id;
            });
            this.logs = merged;
            this.loading = false;
            this.error = null;
            this.cdr.markForCheck();
            return merged;
          }),
        ),
      ),
    );
  }

  readonly eventLabel = eventLabel;
  readonly isErrorEvent = isErrorEvent;
  readonly summary = buildSummary;
}