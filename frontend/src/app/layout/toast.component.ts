import { Component } from '@angular/core';
import { ToastService } from '../services/toast.service';

@Component({
  selector: 'app-toast',
  standalone: true,
  template: `
    <div class="toast-container">
      @for (t of toastService.toasts(); track t.id) {
        <div class="toast {{ t.type }}" [class.leaving]="false">
          <span>{{ t.type === 'success' ? '✅' : t.type === 'error' ? '❌' : 'ℹ️' }}</span>
          <span>{{ t.message }}</span>
        </div>
      }
    </div>
    <style>
      .toast-container {
        position: fixed; top: 1rem; right: 1rem; z-index: 9999;
        display: flex; flex-direction: column; gap: 0.5rem;
      }
      .toast {
        display: flex; align-items: center; gap: 0.5rem;
        padding: 0.7rem 1.2rem; border-radius: 10px;
        font-size: 0.9rem; font-weight: 500;
        box-shadow: 0 4px 16px rgba(0,0,0,0.15);
        animation: slideIn 0.25s ease-out;
        color: #fff; min-width: 250px;
      }
      .toast.success { background: #27ae60; }
      .toast.error   { background: #c0392b; }
      .toast.info    { background: #2980b9; }
      @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to   { transform: translateX(0); opacity: 1; }
      }
    </style>
  `
})
export class ToastComponent {
  constructor(public toastService: ToastService) {}
}
