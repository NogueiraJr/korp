import { Routes } from '@angular/router';
import { LoginComponent } from './features/login/login';
import { ShellComponent } from './features/shell/shell';
import { ProductsComponent } from './features/products/products';
import { InvoicesComponent } from './features/invoices/invoices';
import { LogsComponent } from './features/logs/logs';
import { authGuard } from './core/guards/auth.guard';

export const routes: Routes = [
  { path: 'login', component: LoginComponent },
  {
    path: '',
    component: ShellComponent,
    canActivate: [authGuard],
    children: [
      { path: '', redirectTo: 'products', pathMatch: 'full' },
      { path: 'products', component: ProductsComponent },
      { path: 'invoices', component: InvoicesComponent },
      { path: 'logs', component: LogsComponent },
    ],
  },
  { path: '**', redirectTo: '' },
];