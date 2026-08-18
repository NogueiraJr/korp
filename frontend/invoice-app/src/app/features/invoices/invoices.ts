import { ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Subject, forkJoin, takeUntil } from 'rxjs';
import { MatDialog } from '@angular/material/dialog';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatChipsModule } from '@angular/material/chips';
import { MatTooltipModule } from '@angular/material/tooltip';
import { CommonModule } from '@angular/common';
import { InvoicesService } from '../../core/services/invoices.service';
import { ProductsService } from '../../core/services/products.service';
import { NotificationService } from '../../core/services/notification.service';
import { Invoice } from '../../core/models/invoice.model';
import { Product } from '../../core/models/product.model';
import { InvoiceDialogComponent } from './invoice-dialog';
import { InvoiceDetailComponent } from './invoice-detail';

interface DisplayInvoice extends Invoice {
  displayItems: { product_code: string; description: string; quantity: number }[];
}

@Component({
  selector: 'app-invoices',
  imports: [
    CommonModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatChipsModule,
    MatTooltipModule,
  ],
  templateUrl: './invoices.html',
  styleUrl: './invoices.scss',
})
export class InvoicesComponent implements OnInit, OnDestroy {
  private readonly destroy$ = new Subject<void>();

  invoices: DisplayInvoice[] = [];
  products: Product[] = [];
  loading = false;
  printingId: string | null = null;
  displayedColumns = ['number', 'status', 'items', 'created_at', 'actions'];

  constructor(
    private readonly invoicesService: InvoicesService,
    private readonly productsService: ProductsService,
    private readonly dialog: MatDialog,
    private readonly notify: NotificationService,
    private readonly cdr: ChangeDetectorRef,
  ) {}

  ngOnInit(): void {
    // Angular lifecycle: ngOnInit loads the initial data.
    this.loadData();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  loadData(): void {
    this.loading = true;
    // RxJS: forkJoin loads products and invoices in parallel.
    forkJoin({
      products: this.productsService.list(),
      invoices: this.invoicesService.list(),
    })
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: ({ products, invoices }) => {
          this.products = products;
          this.invoices = invoices.map((inv) => this.enrich(inv, products));
          this.loading = false;
          this.cdr.markForCheck();
        },
        error: (err) => {
          this.loading = false;
          this.cdr.markForCheck();
          this.notify.fromHttpError(err, 'Erro ao carregar notas fiscais');
        },
      });
  }

  private enrich(invoice: Invoice, products: Product[]): DisplayInvoice {
    const byCode = new Map(products.map((p) => [p.code, p]));
    return {
      ...invoice,
      displayItems: invoice.items.map((item) => ({
        product_code: item.product_code,
        description: byCode.get(item.product_code)?.description ?? item.product_code,
        quantity: item.quantity,
      })),
    };
  }

  openCreate(): void {
    const dialogRef = this.dialog.open(InvoiceDialogComponent, {
      width: '560px',
      disableClose: true,
      data: { products: this.products },
    });
    dialogRef.afterClosed().subscribe((saved: boolean) => {
      if (saved) {
        this.notify.success('Nota fiscal criada');
        this.loadData();
      }
      this.cdr.markForCheck();
    });
  }

  openDetail(invoice: DisplayInvoice): void {
    this.dialog.open(InvoiceDetailComponent, {
      width: '640px',
      data: {
        invoice: {
          id: invoice.id,
          number: invoice.number,
          status: invoice.status,
          items: invoice.displayItems,
          created_at: invoice.created_at,
          closed_at: invoice.closed_at,
        },
        products: this.products,
      },
    });
  }

  print(invoice: DisplayInvoice): void {
    if (invoice.status !== 'OPEN' || this.printingId) {
      return;
    }
    this.printingId = invoice.id;
    // The idempotency key guarantees that a network retry (double click or
    // automatic retry) never consumes stock twice.
    const idempotencyKey = crypto.randomUUID();

    this.invoicesService.print(invoice.id, idempotencyKey).subscribe({
      next: (updated) => {
        this.printingId = null;
        this.notify.success(`Nota #${updated.number} fechada e saldos atualizados`);
        this.loadData();
        this.cdr.markForCheck();
      },
      error: (err) => {
        this.printingId = null;
        this.cdr.markForCheck();
        this.notify.fromHttpError(err, 'Erro ao imprimir nota fiscal');
        // Refresh to reflect any state the backend may have changed.
        this.loadData();
      },
    });
  }

  canPrint(invoice: DisplayInvoice): boolean {
    return invoice.status === 'OPEN' && this.printingId === null;
  }
}