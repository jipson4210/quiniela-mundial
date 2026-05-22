import { Component } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [FormsModule],
  template: `
    <div class="login-page">
      <div class="card">
        <h1>⚽ Quiniela Mundial 2026</h1>

        @if (!isRegister) {
          <h2>Iniciar sesión</h2>
          <form (submit)="onLogin()">
            <label>Email</label>
            <input type="email" [(ngModel)]="email" name="email" required placeholder="tu@email.com" />
            <label>Contraseña</label>
            <input type="password" [(ngModel)]="password" name="password" required placeholder="••••••••" />
            @if (error) { <p class="error">{{ error }}</p> }
            <button type="submit" class="btn-primary" [disabled]="loading">
              {{ loading ? 'Entrando...' : 'Entrar' }}
            </button>
          </form>
          <p class="toggle">¿No tienes cuenta? <a (click)="isRegister=true; error=''">Regístrate</a></p>
        } @else {
          <h2>Crear cuenta</h2>
          <form (submit)="onRegister()">
            <label>Nombre</label>
            <input [(ngModel)]="displayName" name="displayName" required placeholder="Tu nombre" />
            <label>Email</label>
            <input type="email" [(ngModel)]="email" name="email2" required placeholder="tu@email.com" />
            <label>Contraseña</label>
            <input type="password" [(ngModel)]="password" name="password2" required placeholder="mín. 8 caracteres" />
            @if (error) { <p class="error">{{ error }}</p> }
            <button type="submit" class="btn-primary" [disabled]="loading">
              {{ loading ? 'Creando...' : 'Registrarse' }}
            </button>
          </form>
          <p class="toggle">¿Ya tienes cuenta? <a (click)="isRegister=false; error=''">Inicia sesión</a></p>
        }
      </div>
    </div>

    <style>
      .login-page {
        display: flex; align-items: center; justify-content: center;
        min-height: 100vh; background: var(--color-bg);
      }
      .card {
        background: var(--color-surface); padding: 2.5rem; border-radius: 12px;
        box-shadow: 0 4px 24px rgba(0,0,0,0.08); width: 100%; max-width: 420px;
      }
      h1 { text-align: center; font-size: 1.4rem; color: var(--color-primary); margin-bottom: 0.5rem; }
      h2 { text-align: center; font-size: 1.1rem; margin-bottom: 1.2rem; color: var(--color-text); }
      label { display: block; font-size: 0.85rem; margin: 0.8rem 0 0.25rem; color: var(--color-text-secondary); }
      input {
        width: 100%; padding: 0.65rem; border: 1px solid var(--color-border);
        border-radius: 8px; background: var(--color-bg); color: var(--color-text); font-size: 0.95rem;
      }
      .btn-primary {
        width: 100%; margin-top: 1.2rem; padding: 0.7rem;
        background: var(--color-primary); color: #fff; border: none;
        border-radius: 8px; font-size: 1rem; font-weight: 600; cursor: pointer;
      }
      .btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
      .error { color: var(--color-danger); font-size: 0.85rem; margin-top: 0.5rem; }
      .toggle { text-align: center; margin-top: 1rem; font-size: 0.85rem; color: var(--color-text-secondary); }
      .toggle a { color: var(--color-primary); cursor: pointer; text-decoration: underline; }
    </style>
  `
})
export class LoginComponent {
  isRegister = false;
  email = '';
  password = '';
  displayName = '';
  error = '';
  loading = false;

  constructor(private auth: AuthService, private router: Router) {}

  onLogin() {
    if (!this.email || !this.password) { this.error = 'Completa todos los campos.'; return; }
    this.loading = true; this.error = '';
    this.auth.login(this.email, this.password).subscribe({
      next: (res) => { this.auth.saveSession(res); this.router.navigate(['/dashboard']); },
      error: () => { this.error = 'Credenciales inválidas.'; this.loading = false; }
    });
  }

  onRegister() {
    if (!this.email || !this.password || !this.displayName) { this.error = 'Completa todos los campos.'; return; }
    if (this.password.length < 8) { this.error = 'La contraseña debe tener al menos 8 caracteres.'; return; }
    this.loading = true; this.error = '';
    this.auth.register(this.email, this.password, this.displayName).subscribe({
      next: (res) => { this.auth.saveSession(res); this.router.navigate(['/dashboard']); },
      error: (err) => { this.error = err.error?.error || 'Error al registrarse.'; this.loading = false; }
    });
  }
}
