import { Routes } from '@angular/router';

export const routes: Routes = [
  { path: '', redirectTo: '/dashboard', pathMatch: 'full' },
  // Routes will be added in later phases:
  // { path: 'dashboard', loadComponent: () => import('./pages/dashboard/dashboard.component') },
  // { path: 'pools/:id', loadComponent: () => import('./pages/pool/pool.component') },
  // { path: 'login', loadComponent: () => import('./pages/login/login.component') },
];
