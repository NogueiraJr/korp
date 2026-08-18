import { ChangeDetectorRef, Component, Inject, OnInit } from '@angular/core';
import {
  FormArray,
  FormBuilder,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { CommonModule } from '@angular/common';
import { InvoicesService } from '../../core/services/invoices.service';
import { NotificationService } from '../../core/services/notification.service';
import { Product } from '../../core/models/product.model';

export interface InvoiceDialogData {
  products: Product[];
}

@Component({
  selector: 'app-invoice-dialog',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
  ],
  templateUrl: './invoice-dialog.html',
  styleUrl: './invoice-dialog.scss',
})
export class InvoiceDialogComponent implements OnInit {
  form: FormGroup;
  saving = false;
  products: Product[];

  constructor(
    @Inject(MAT_DIALOG_DATA) private readonly data: InvoiceDialogData,
    readonly dialogRef: MatDialogRef<InvoiceDialogComponent>,
    private readonly fb: FormBuilder,
    private readonly invoicesService: InvoicesService,
    private readonly notify: NotificationService,
    private readonly cdr: ChangeDetectorRef,
  ) {
    this.products = data?.products ?? [];
    this.form = this.fb.group({
      items: this.fb.array([]),
    });
  }

  ngOnInit(): void {
    // Angular lifecycle: ngOnInit seeds the first item row.
    this.addItem();
  }

  get items(): FormArray {
    return this.form.get('items') as FormArray;
  }

  addItem(): void {
    this.items.push(
      this.fb.group({
        product_code: ['', Validators.required],
        quantity: [1, [Validators.required, Validators.min(1)]],
      }),
    );
  }

  removeItem(index: number): void {
    this.items.removeAt(index);
  }

  productDescription(code: string): string {
    return this.products.find((p) => p.code === code)?.description ?? code;
  }

  maxQuantity(code: string): number {
    return this.products.find((p) => p.code === code)?.stock_quantity ?? 0;
  }

  save(): void {
    const payload = { items: [] as { product_code: string; quantity: number }[] };
    const selected = new Set<string>();

    for (const row of this.items.controls) {
      const code = row.value.product_code;
      const quantity = Number(row.value.quantity);
      if (!code || !quantity || quantity <= 0) {
        this.form.markAllAsTouched();
        this.notify.error('Preencha todos os itens com produto e quantidade válida');
        return;
      }
      if (selected.has(code)) {
        this.notify.error(`O produto "${code}" está duplicado na nota`);
        return;
      }
      selected.add(code);
      payload.items.push({ product_code: code, quantity });
    }

    if (payload.items.length === 0) {
      this.notify.error('Adicione ao menos um produto à nota');
      return;
    }

    this.saving = true;
    this.invoicesService.create(payload).subscribe({
      next: (invoice) => {
        this.saving = false;
        this.cdr.markForCheck();
        this.notify.success(`Nota #${invoice.number} criada com status Aberta`);
        this.dialogRef.close(true);
      },
      error: (err) => {
        this.saving = false;
        this.cdr.markForCheck();
        this.notify.fromHttpError(err, 'Erro ao criar nota fiscal');
      },
    });
  }
}