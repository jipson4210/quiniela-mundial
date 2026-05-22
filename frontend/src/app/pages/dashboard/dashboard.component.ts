import { Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { ApiService, Pool } from '../../services/api.service';
import { AuthService } from '../../services/auth.service';
import { NavbarComponent } from '../../layout/navbar.component';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [FormsModule, RouterModule, NavbarComponent],
  template: `
    <app-navbar />
    <div class="container">
      <div class="header">
        <h1>Mis Quinielas</h1>
        <button class="btn-primary" (click)="showCreate = !showCreate">
          {{ showCreate ? 'Cancelar' : '+ Nuevo Grupo' }}
        </button>
      </div>

      @if (showCreate) {
        <div class="card create-form">
          <h3>Crear nuevo grupo</h3>
          <form (submit)="onCreate()">
            <label>Nombre del grupo</label>
            <input [(ngModel)]="newPoolName" name="name" required placeholder="Ej: La Quiniela Pro" />
            <label>Descripción</label>
            <input [(ngModel)]="newPoolDesc" name="desc" placeholder="Opcional" />
            <label>ID del Torneo</label>
            <input [(ngModel)]="tournamentId" name="tid" required />
            @if (createError) { <p class="error">{{ createError }}</p> }
            <button type="submit" class="btn-primary" [disabled]="creating">
              {{ creating ? 'Creando...' : 'Crear Grupo' }}
            </button>
          </form>
        </div>
      }

      @if (loading) {
        <p class="loading">Cargando grupos...</p>
      } @else if (pools.length === 0) {
        <div class="empty">
          <p>No tienes grupos todavía. ¡Crea uno o acepta una invitación!</p>
        </div>
      } @else {
        <div class="pool-grid">
          @for (pool of pools; track pool.PoolID) {
            <div class="card pool-card">
              <h3>{{ pool.Name }}</h3>
              <a [routerLink]="['/pools', pool.PoolID]">Entrar →</a>
            </div>
          }
        </div>
      }

      <div class="card invite-section">
        <h3>Aceptar invitación</h3>
        <form (submit)="onAcceptInvite()">
          <input [(ngModel)]="inviteToken" name="token" placeholder="Pega aquí el token de invitación" />
          <button type="submit" class="btn-secondary">Unirme</button>
        </form>
        @if (inviteError) { <p class="error">{{ inviteError }}</p> }
        @if (inviteSuccess) { <p class="success">{{ inviteSuccess }}</p> }
      </div>
    </div>

    <style>
      .container { max-width: 900px; margin: 0 auto; padding: 2rem 1rem; }
      .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
      h1 { color: var(--color-text); font-size: 1.5rem; }
      .card { background: var(--color-surface); border-radius: 12px; padding: 1.5rem; margin-bottom: 1rem; box-shadow: 0 2px 12px rgba(0,0,0,0.04); }
      .create-form label { display: block; font-size: 0.85rem; margin-top: 0.8rem; color: var(--color-text-secondary); }
      .create-form input { width: 100%; padding: 0.55rem; border: 1px solid var(--color-border); border-radius: 8px; background: var(--color-bg); color: var(--color-text); }
      .btn-primary { background: var(--color-primary); color: #fff; border: none; padding: 0.6rem 1.2rem; border-radius: 8px; font-weight: 600; cursor: pointer; }
      .btn-secondary { background: var(--color-accent); color: #fff; border: none; padding: 0.5rem 1rem; border-radius: 8px; font-weight: 600; cursor: pointer; }
      .pool-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(250px, 1fr)); gap: 1rem; }
      .pool-card h3 { margin-bottom: 0.5rem; }
      .pool-card a { color: var(--color-primary); text-decoration: none; font-weight: 600; }
      .invite-section { display: flex; gap: 1rem; align-items: flex-end; flex-wrap: wrap; }
      .invite-section input { flex: 1; min-width: 200px; padding: 0.55rem; border: 1px solid var(--color-border); border-radius: 8px; background: var(--color-bg); color: var(--color-text); }
      .loading, .empty { text-align: center; color: var(--color-text-secondary); padding: 3rem; }
      .error { color: var(--color-danger); font-size: 0.85rem; }
      .success { color: var(--color-success); font-size: 0.85rem; }
    </style>
  `
})
export class DashboardComponent implements OnInit {
  pools: Pool[] = [];
  loading = true;
  showCreate = false;
  newPoolName = ''; newPoolDesc = ''; tournamentId = '';
  creating = false; createError = '';
  inviteToken = ''; inviteError = ''; inviteSuccess = '';

  constructor(private api: ApiService, private auth: AuthService) {}

  ngOnInit() {
    this.loadPools();
  }

  loadPools() {
    this.loading = true;
    this.api.getPools().subscribe({
      next: (res) => { this.pools = res.pools || []; this.loading = false; },
      error: () => { this.loading = false; }
    });
  }

  onCreate() {
    if (!this.newPoolName || !this.tournamentId) { this.createError = 'Completa los campos.'; return; }
    this.creating = true; this.createError = '';
    this.api.createPool(this.newPoolName, this.newPoolDesc, this.tournamentId).subscribe({
      next: () => {
        this.showCreate = false; this.newPoolName = ''; this.newPoolDesc = '';
        this.tournamentId = ''; this.creating = false; this.loadPools();
      },
      error: (err) => { this.createError = err.error?.error || 'Error'; this.creating = false; }
    });
  }

  onAcceptInvite() {
    if (!this.inviteToken) { this.inviteError = 'Pega el token.'; return; }
    this.inviteError = ''; this.inviteSuccess = '';
    this.api.acceptInvitation(this.inviteToken).subscribe({
      next: () => { this.inviteSuccess = '¡Te uniste al grupo!'; this.inviteToken = ''; this.loadPools(); },
      error: (err) => { this.inviteError = err.error?.error || 'Token inválido o expirado.'; }
    });
  }
}
