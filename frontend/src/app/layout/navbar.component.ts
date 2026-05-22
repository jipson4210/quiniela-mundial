import { Component } from '@angular/core';
import { RouterModule } from '@angular/router';
import { AuthService } from '../services/auth.service';
import { ThemeSwitcherComponent } from './theme-switcher.component';

@Component({
  selector: 'app-navbar',
  standalone: true,
  imports: [RouterModule, ThemeSwitcherComponent],
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

      <app-theme-switcher />
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
    </style>
  `,
})
export class NavbarComponent {
  constructor(public auth: AuthService) {}
}
