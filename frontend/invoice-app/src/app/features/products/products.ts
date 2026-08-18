import { ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Subject, takeUntil } from 'rxjs';
import { MatDialog } from '@angular/material/dialog';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatChipsModule } from '@angular/material/chips';
import { MatTooltipModule } from '@angular/material/tooltip';
import { CommonModule } from '@angular/common';
import { ProductsService } from '../../core/services/products.service';
import { NotificationService } from '../../core/services/notification.service';
import { Product } from '../../core/models/product.model';
import { ProductDialogComponent } from './product-dialog';
import { ProductDetailComponent } from './product-detail';

@Component({
  selector: 'app-products',
  imports: [
    CommonModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatProgressBarModule,
    MatChipsModule,
    MatTooltipModule,
  ],
  templateUrl: './products.html',
  styleUrl: './products.scss',
})
export class ProductsComponent implements OnInit, OnDestroy {
  private readonly destroy$ = new Subject<void>();

  products: Product[] = [];
  loading = false;
  displayedColumns = ['code', 'description', 'stock_quantity', 'created_at', 'actions'];

  constructor(
    private readonly productsService: ProductsService,
    private readonly dialog: MatDialog,
    private readonly notify: NotificationService,
    private readonly cdr: ChangeDetectorRef,
  ) {}

  ngOnInit(): void {
    // Angular lifecycle: ngOnInit loads the initial product list.
    this.loadProducts();
  }

  ngOnDestroy(): void {
    // Angular lifecycle: ngOnDestroy unsubscribes to prevent memory leaks.
    this.destroy$.next();
    this.destroy$.complete();
  }

  loadProducts(): void {
    this.loading = true;
    this.productsService
      .list()
      .pipe(takeUntil(this.destroy$))
      .subscribe({
        next: (products) => {
          this.products = products;
          this.loading = false;
          this.cdr.markForCheck();
        },
        error: (err) => {
          this.loading = false;
          this.cdr.markForCheck();
          this.notify.fromHttpError(err, 'Erro ao carregar produtos');
        },
      });
  }

  openCreate(): void {
    const dialogRef = this.dialog.open(ProductDialogComponent, {
      width: '480px',
      disableClose: true,
    });
    dialogRef.afterClosed().subscribe((saved: boolean) => {
      if (saved) {
        this.notify.success('Produto cadastrado com sucesso');
        this.loadProducts();
      }
      this.cdr.markForCheck();
    });
  }

  openEdit(product: Product): void {
    const dialogRef = this.dialog.open(ProductDialogComponent, {
      width: '480px',
      disableClose: true,
      data: product,
    });
    dialogRef.afterOpened().subscribe(() => this.cdr.markForCheck());
    dialogRef.afterClosed().subscribe((saved: boolean) => {
      if (saved) {
        this.notify.success('Produto atualizado com sucesso');
        this.loadProducts();
      }
      this.cdr.markForCheck();
    });
  }

  openDetail(product: Product): void {
    this.dialog.open(ProductDetailComponent, {
      width: '560px',
      data: { product },
    });
  }
}