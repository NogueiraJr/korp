import { ChangeDetectorRef, Component, Inject, OnDestroy, OnInit } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialog, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { forkJoin, Subject, takeUntil } from 'rxjs';
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatChipsModule } from '@angular/material/chips';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Product } from '../../core/models/product.model';
import { ActivityLog } from '../../core/models/api.model';
import { LogsService } from '../../core/services/logs.service';
import { ProductsService } from '../../core/services/products.service';
import { buildSummary, eventLabel, isErrorEvent } from '../../core/utils/activity-log.util';
import { InvoiceDetailComponent } from '../invoices/invoice-detail';

export interface ProductDetailData {
  product: Product;
}

interface InvoiceUsage {
  number: number;
  quantity: number;
  closed: boolean;
  created_at: string;
}

@Component({
  selector: 'app-product-detail',
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatChipsModule,
    MatProgressBarModule,
    MatTableModule,
    MatTooltipModule,
  ],
  templateUrl: './product-detail.html',
  styleUrl: './product-detail.scss',
})
export class ProductDetailComponent implements OnInit, OnDestroy {
  private readonly destroy$ = new Subject<void>();

  product: Product;
  history: ActivityLog[] = [];
  usages: InvoiceUsage[] = [];
  loadingData = false;

  displayedColumns = ['number', 'quantity', 'status', 'created_at'];

  readonly eventLabel = eventLabel;
  readonly isErrorEvent = isErrorEvent;
  readonly summary = buildSummary;

  private faturamentoLogs: ActivityLog[] = [];

  constructor(
    @Inject(MAT_DIALOG_DATA) private readonly data: ProductDetailData,
    readonly dialogRef: MatDialogRef<ProductDetailComponent>,
    private readonly logsService: LogsService,
    private readonly productsService: ProductsService,
    private readonly dialog: MatDialog,
    private readonly cdr: ChangeDetectorRef,
  ) {
    this.product = data.product;
  }

  ngOnInit(): void {
    // Angular lifecycle: ngOnInit loads the product history and the invoices
    // that consumed this product (used in processing).
    this.loadingData = true;
    forkJoin({
      estoque: this.logsService.fetch('estoque'),
      faturamento: this.logsService.fetch('faturamento'),
    })
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: ({ estoque, faturamento }) => {
          this.faturamentoLogs = faturamento;
          this.history = estoque.filter((log) => {
            const d = log.details;
            if (!d) {
              return false;
            }
            if (d['code'] === this.product.code) {
              return true;
            }
            // Stock events store the affected codes in the items array.
            return (
              Array.isArray(d['items']) &&
              d['items'].some((it) => (it as { code?: string }).code === this.product.code)
            );
          });
          this.usages = this.buildUsages(faturamento);
          this.loadingData = false;
          this.cdr.markForCheck();
        },
        error: () => {
          this.loadingData = false;
          this.cdr.markForCheck();
        },
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  private buildUsages(logs: ActivityLog[]): InvoiceUsage[] {
    const map = new Map<number, InvoiceUsage>();
    for (const log of logs) {
      const d = log.details;
      if (!d || !Array.isArray(d['items']) || typeof d['number'] !== 'number') {
        continue;
      }
      const number = d['number'] as number;
      if (log.event === 'invoice.created') {
        const quantity = (d['items'] as { product_code?: string; quantity?: number }[])
          .filter((it) => it.product_code === this.product.code)
          .reduce((sum, it) => sum + (it.quantity ?? 0), 0);
        if (quantity > 0 && !map.has(number)) {
          map.set(number, { number, quantity, closed: false, created_at: log.created_at });
        }
      } else if (log.event === 'invoice.print_success') {
        const usage = map.get(number);
        if (usage) {
          usage.closed = true;
        }
      }
    }
    return [...map.values()].sort((a, b) => b.created_at.localeCompare(a.created_at));
  }

  openInvoice(usage: InvoiceUsage): void {
    // RxJS: loads the product catalog to enrich the invoice items with
    // descriptions before opening the invoice detail dialog.
    this.productsService.list().pipe(takeUntil(this.destroy$)).subscribe({
      next: (products) => {
        this.cdr.markForCheck();
        const byCode = new Map(products.map((p) => [p.code, p]));
        const created = this.faturamentoLogs.find(
          (log) => log.event === 'invoice.created' && log.details?.['number'] === usage.number,
        );
        const items = ((created?.details?.['items'] as { product_code?: string; quantity?: number }[]) ?? []).map(
          (it) => ({
            product_code: it.product_code ?? '',
            description: byCode.get(it.product_code ?? '')?.description ?? it.product_code ?? '',
            quantity: it.quantity ?? 0,
          }),
        );
        this.dialog.open(InvoiceDetailComponent, {
          width: '640px',
          data: {
            invoice: {
              id: '',
              number: usage.number,
              status: usage.closed ? 'CLOSED' : 'OPEN',
              items,
              created_at: usage.created_at,
              closed_at: null,
            },
            products,
          },
        });
      },
    });
  }
}