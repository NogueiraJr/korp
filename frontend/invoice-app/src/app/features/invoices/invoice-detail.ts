import { ChangeDetectorRef, Component, Inject, OnDestroy, OnInit } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialog, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { Subject, takeUntil } from 'rxjs';
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatChipsModule } from '@angular/material/chips';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { ActivityLog } from '../../core/models/api.model';
import { Product } from '../../core/models/product.model';
import { LogsService } from '../../core/services/logs.service';
import { buildSummary, eventLabel, isErrorEvent } from '../../core/utils/activity-log.util';
import { ProductDetailComponent } from '../products/product-detail';

export interface InvoiceDetailData {
  invoice: {
    id: string;
    number: number;
    status: 'OPEN' | 'CLOSED';
    items: { product_code: string; description: string; quantity: number }[];
    created_at: string;
    closed_at?: string | null;
  };
  products?: Product[];
}

@Component({
  selector: 'app-invoice-detail',
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatChipsModule,
    MatProgressBarModule,
    MatTooltipModule,
  ],
  templateUrl: './invoice-detail.html',
  styleUrl: './invoice-detail.scss',
})
export class InvoiceDetailComponent implements OnInit, OnDestroy {
  private readonly destroy$ = new Subject<void>();

  invoice: InvoiceDetailData['invoice'];
  products: Product[] = [];
  history: ActivityLog[] = [];
  loadingHistory = false;

  readonly eventLabel = eventLabel;
  readonly isErrorEvent = isErrorEvent;
  readonly summary = buildSummary;

  constructor(
    @Inject(MAT_DIALOG_DATA) private readonly data: InvoiceDetailData,
    readonly dialogRef: MatDialogRef<InvoiceDetailComponent>,
    private readonly logsService: LogsService,
    private readonly dialog: MatDialog,
    private readonly cdr: ChangeDetectorRef,
  ) {
    this.invoice = data.invoice;
    this.products = data.products ?? [];
  }

  ngOnInit(): void {
    // Angular lifecycle: ngOnInit loads the invoice history (creation/print).
    this.loadingHistory = true;
    this.logsService
      .fetch('faturamento')
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (logs) => {
          this.history = logs.filter((log) => log.details?.['number'] === this.invoice.number);
          this.loadingHistory = false;
          this.cdr.markForCheck();
        },
        error: () => {
          this.loadingHistory = false;
          this.cdr.markForCheck();
        },
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  get totalQuantity(): number {
    return this.invoice.items.reduce((sum, item) => sum + item.quantity, 0);
  }

  openProduct(productCode: string): void {
    const product = this.products.find((p) => p.code === productCode);
    if (product) {
      this.dialog.open(ProductDetailComponent, {
        width: '560px',
        data: { product },
      });
    }
  }
}