import { Component, inject } from '@angular/core';
import { ThemeService } from '../services/theme.service';

@Component({
  selector: 'app-theme-switcher',
  standalone: true,
  template: `
    <div class="theme-picker" role="group" aria-label="Cambiar tema visual">
      <span class="theme-label">Tema:</span>
      @for (t of theme.available; track t.id) {
        <button
          type="button"
          class="theme-dot {{ t.id }}"
          [class.active]="theme.current() === t.id"
          [attr.aria-pressed]="theme.current() === t.id"
          [attr.title]="t.label + ' — ' + t.description"
          (click)="theme.setTheme(t.id)">
        </button>
      }
    </div>

    <style>
      .theme-picker { display: flex; gap: 0.4rem; align-items: center; }
      .theme-label { font-size: 0.75rem; opacity: 0.7; }
      .theme-dot {
        width: 20px; height: 20px; border-radius: 50%;
        border: 2px solid transparent; cursor: pointer;
        transition: border-color 0.2s, transform 0.15s;
        padding: 0;
      }
      .theme-dot:hover { transform: scale(1.08); }
      .theme-dot.active { border-color: #fff; }
      .mundial-vibrante { background: linear-gradient(135deg, #1a936f, #f39c12); }
      .dark-pro { background: #0d1117; border-color: #30363d; }
      .latina-calida { background: linear-gradient(135deg, #d62828, #fcbf49); }
    </style>
  `,
})
export class ThemeSwitcherComponent {
  theme = inject(ThemeService);
}
