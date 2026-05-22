import { Component } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AuthService } from '../services/auth.service';

type Theme = 'mundial-vibrante' | 'dark-pro' | 'latina-calida';

@Component({
  selector: 'app-navbar',
  standalone: true,
  imports: [RouterModule],
  template: `
    <nav class="navbar">
      <a routerLink="/dashboard" class="brand">⚽ Quiniela 2026</a>

      <div class="nav-links">
        @if (auth.isLoggedIn()) {
          <a routerLink="/dashboard">Mis Grupos</a>
          <span class="user-info">{{ auth.user()?.DisplayName }}</span>
          <button class="btn-text" (click)="auth.logout()">Salir</button>
        }
      </div>

      <div class="theme-picker">
        <span class="theme-label">Tema:</span>
        @for (t of themes; track t) {
          <button class="theme-dot {{t}}"
                  [class.active]="currentTheme === t"
                  [attr.title]="themeNames[t]"
                  (click)="setTheme(t)"></button>
        }
      </div>
    </nav>

    <style>
      .navbar {
        display: flex;
        align-items: center;
        gap: 1rem;
        padding: 0.6rem 1.2rem;
        background: var(--color-header-bg);
        color: var(--color-header-text);
        font-family: var(--font-heading);
      }
      .brand {
        font-size: 1.2rem;
        font-weight: 700;
        text-decoration: none;
        color: inherit;
        margin-right: auto;
      }
      .nav-links { display: flex; gap: 1rem; align-items: center; }
      .nav-links a { color: inherit; text-decoration: none; opacity: 0.9; }
      .nav-links a:hover { opacity: 1; }
      .user-info { font-size: 0.85rem; opacity: 0.8; }
      .btn-text {
        background: none; border: 1px solid rgba(255,255,255,0.3); color: inherit;
        padding: 0.3rem 0.6rem; border-radius: 4px; cursor: pointer; font-size: 0.8rem;
      }
      .theme-picker { display: flex; gap: 0.4rem; align-items: center; }
      .theme-label { font-size: 0.75rem; opacity: 0.7; }
      .theme-dot {
        width: 20px; height: 20px; border-radius: 50%;
        border: 2px solid transparent; cursor: pointer; transition: border 0.2s;
      }
      .theme-dot.active { border-color: #fff; }
      .mundial-vibrante { background: linear-gradient(135deg, #1a936f, #f39c12); }
      .dark-pro { background: #0d1117; border-color: #30363d; }
      .latina-calida { background: linear-gradient(135deg, #d62828, #fcbf49); }
    </style>
  `
})
export class NavbarComponent {
  themes: Theme[] = ['mundial-vibrante', 'dark-pro', 'latina-calida'];
  themeNames: Record<Theme, string> = {
    'mundial-vibrante': 'Mundial Vibrante',
    'dark-pro': 'Dark Pro',
    'latina-calida': 'Latina Cálida'
  };
  currentTheme: Theme = 'mundial-vibrante';

  constructor(public auth: AuthService) {
    const saved = localStorage.getItem('theme') as Theme;
    if (saved && this.themes.includes(saved)) {
      this.currentTheme = saved;
    }
    this.applyTheme(this.currentTheme);
  }

  setTheme(t: Theme) {
    this.currentTheme = t;
    localStorage.setItem('theme', t);
    this.applyTheme(t);
  }

  private applyTheme(t: Theme) {
    document.documentElement.setAttribute('data-theme', t);
  }
}
