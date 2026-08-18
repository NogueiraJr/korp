import { ChangeDetectorRef, Component, Inject, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { CommonModule } from '@angular/common';
import { ProductsService } from '../../core/services/products.service';
import { NotificationService } from '../../core/services/notification.service';
import { Product } from '../../core/models/product.model';

export interface ProductDialogData {
  product?: Product;
}

@Component({
  selector: 'app-product-dialog',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
  ],
  templateUrl: './product-dialog.html',
  styleUrl: './product-dialog.scss',
})
export class ProductDialogComponent implements OnInit {
  form: FormGroup;
  isEdit: boolean;
  saving = false;
  suggesting = false;

  constructor(
    @Inject(MAT_DIALOG_DATA) private readonly data: ProductDialogData,
    readonly dialogRef: MatDialogRef<ProductDialogComponent>,
    private readonly fb: FormBuilder,
    private readonly productsService: ProductsService,
    private readonly notify: NotificationService,
    private readonly cdr: ChangeDetectorRef,
  ) {
    this.isEdit = !!data?.product;
    this.form = this.fb.group({
      code: [data?.product?.code ?? '', [Validators.required]],
      description: [data?.product?.description ?? '', [Validators.required]],
      stock_quantity: [
        data?.product?.stock_quantity ?? 0,
        [Validators.required, Validators.min(0)],
      ],
    });

    // Desabilita o campo código apenas na visualização (modo editar),
    // mas permite alteração no modo criação.
    if (this.isEdit) {
      this.form.get('code')?.disable();
    } else {
      this.form.get('code')?.enable();
    }
  }

  ngOnInit(): void {
    // Garante o change detection no modo zoneless do Angular 22.
    this.cdr.markForCheck();

    // Em casos race conditions de diálogo, garante que os valores do produto
    // estejam visíveis no formulário aplicando-os novamente via patchValue.
    if (this.form.get('code')?.value === '' && this.data?.product) {
      this.form.patchValue({
        code: this.data.product.code,
        description: this.data.product.description,
        stock_quantity: this.data.product.stock_quantity,
      });
      this.cdr.markForCheck();
    }

    if (!this.isEdit) {
      this.form.controls['code'].valueChanges.subscribe((value) => {
        // Clear a stale AI suggestion when the code changes.
        if (this.form.controls['description'].value?.startsWith('Produto ')) {
          this.form.controls['description'].setValue('');
        }
      });
    }
  }

  suggestWithAI(): void {
    const code = this.form.value.code?.trim();
    if (!code) {
      this.notify.error('Informe o código antes de usar a IA');
      return;
    }
    this.suggesting = true;
    this.productsService.suggestDescription(code).subscribe({
      next: (res) => {
        this.suggesting = false;
        this.form.controls['description'].setValue(res.suggestion);
        this.cdr.markForCheck();
        this.notify.success('Descrição sugerida pela IA');
      },
      error: () => {
        this.suggesting = false;
        this.cdr.markForCheck();
        this.notify.error('Não foi possível obter sugestão da IA');
      },
    });
  }

  save(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }
    this.saving = true;

    const payload = {
      code: this.form.value.code?.trim(),
      description: this.form.value.description?.trim(),
      stock_quantity: Number(this.form.value.stock_quantity),
    };

    const request = this.isEdit
      ? this.productsService.update(payload.code, {
          description: payload.description,
          stock_quantity: payload.stock_quantity,
        })
      : this.productsService.create(payload);

    request.subscribe({
      next: () => {
        this.saving = false;
        this.cdr.markForCheck();
        this.dialogRef.close(true);
      },
      error: (err) => {
        this.saving = false;
        this.cdr.markForCheck();
        this.notify.fromHttpError(err, 'Erro ao salvar produto');
      },
    });
  }
}