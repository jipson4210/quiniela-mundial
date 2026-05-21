---
name: angular-feature
description: How to create Angular standalone features (components, services, routes, forms) for the Quiniela Mundial frontend. Use this skill whenever building or modifying anything in /frontend/, when creating new pages or components, when adding routes, when working with reactive forms, or when integrating with the backend API. Apply this for ALL frontend work using Angular 17+ standalone components.
---

# Angular Feature Pattern

Cómo estructurar features en Angular 17+ con **standalone components** (sin NgModules).

## Estructura del proyecto

```
frontend/
├── src/
│   ├── app/
│   │   ├── core/                       # Singletons globales
│   │   │   ├── auth/
│   │   │   │   ├── auth.service.ts
│   │   │   │   ├── auth.guard.ts
│   │   │   │   └── auth.interceptor.ts
│   │   │   ├── api/
│   │   │   │   ├── api-client.service.ts
│   │   │   │   └── http-error.interceptor.ts
│   │   │   └── theme/
│   │   │       └── theme.service.ts
│   │   ├── features/                   # Features por dominio
│   │   │   ├── auth/
│   │   │   │   ├── login/
│   │   │   │   │   ├── login.component.ts
│   │   │   │   │   ├── login.component.html
│   │   │   │   │   └── login.component.scss
│   │   │   │   └── register/
│   │   │   ├── pools/
│   │   │   │   ├── pool-list/
│   │   │   │   ├── pool-detail/
│   │   │   │   ├── pool-create/
│   │   │   │   ├── pool-invite/
│   │   │   │   └── pools.routes.ts
│   │   │   ├── predictions/
│   │   │   │   ├── match-predictions/
│   │   │   │   └── bracket-prediction/
│   │   │   └── ranking/
│   │   ├── shared/                     # Reutilizable (componentes, pipes, directives)
│   │   │   ├── components/
│   │   │   │   ├── team-flag/
│   │   │   │   ├── match-card/
│   │   │   │   └── countdown/
│   │   │   ├── pipes/
│   │   │   └── directives/
│   │   ├── styles/
│   │   │   ├── themes/                 # Las 3 paletas (ver theme-system skill)
│   │   │   │   ├── _vibrant.scss
│   │   │   │   ├── _dark-pro.scss
│   │   │   │   └── _latin-warm.scss
│   │   │   ├── _tokens.scss            # Variables CSS globales
│   │   │   └── styles.scss             # Entry point
│   │   ├── app.component.ts
│   │   ├── app.config.ts
│   │   └── app.routes.ts
│   ├── environments/
│   └── main.ts
└── angular.json
```

## Bootstrap moderno (standalone)

```ts
// src/main.ts
import { bootstrapApplication } from '@angular/platform-browser';
import { AppComponent } from './app/app.component';
import { appConfig } from './app/app.config';

bootstrapApplication(AppComponent, appConfig)
  .catch(err => console.error(err));
```

```ts
// src/app/app.config.ts
import { ApplicationConfig, provideZoneChangeDetection } from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { routes } from './app.routes';
import { authInterceptor } from './core/auth/auth.interceptor';
import { httpErrorInterceptor } from './core/api/http-error.interceptor';

export const appConfig: ApplicationConfig = {
  providers: [
    provideZoneChangeDetection({ eventCoalescing: true }),
    provideRouter(routes),
    provideHttpClient(
      withInterceptors([authInterceptor, httpErrorInterceptor])
    ),
    provideAnimationsAsync(),
  ],
};
```

## Routing con lazy loading

```ts
// src/app/app.routes.ts
import { Routes } from '@angular/router';
import { authGuard } from './core/auth/auth.guard';

export const routes: Routes = [
  { path: '', redirectTo: 'pools', pathMatch: 'full' },
  {
    path: 'login',
    loadComponent: () => import('./features/auth/login/login.component')
      .then(m => m.LoginComponent),
  },
  {
    path: 'register',
    loadComponent: () => import('./features/auth/register/register.component')
      .then(m => m.RegisterComponent),
  },
  {
    path: 'pools',
    canActivate: [authGuard],
    loadChildren: () => import('./features/pools/pools.routes')
      .then(m => m.poolsRoutes),
  },
  {
    path: 'predictions',
    canActivate: [authGuard],
    loadChildren: () => import('./features/predictions/predictions.routes')
      .then(m => m.predictionsRoutes),
  },
];
```

```ts
// src/app/features/pools/pools.routes.ts
import { Routes } from '@angular/router';

export const poolsRoutes: Routes = [
  {
    path: '',
    loadComponent: () => import('./pool-list/pool-list.component')
      .then(m => m.PoolListComponent),
  },
  {
    path: 'create',
    loadComponent: () => import('./pool-create/pool-create.component')
      .then(m => m.PoolCreateComponent),
  },
  {
    path: ':id',
    loadComponent: () => import('./pool-detail/pool-detail.component')
      .then(m => m.PoolDetailComponent),
  },
];
```

## Componente standalone con Signals

```ts
// src/app/features/pools/pool-list/pool-list.component.ts
import { Component, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

import { PoolsService, Pool } from '../pools.service';

@Component({
  selector: 'app-pool-list',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './pool-list.component.html',
  styleUrl: './pool-list.component.scss',
})
export class PoolListComponent {
  private poolsService = inject(PoolsService);

  pools = signal<Pool[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);

  poolsByRole = computed(() => {
    const all = this.pools();
    return {
      asCreator: all.filter(p => p.role === 'creator'),
      asAdmin:   all.filter(p => p.role === 'admin'),
      asMember:  all.filter(p => p.role === 'member'),
    };
  });

  constructor() {
    this.load();
  }

  load() {
    this.loading.set(true);
    this.poolsService.list().subscribe({
      next: pools => {
        this.pools.set(pools);
        this.loading.set(false);
      },
      error: err => {
        this.error.set('No se pudieron cargar las quinielas');
        this.loading.set(false);
      },
    });
  }
}
```

```html
<!-- pool-list.component.html -->
<div class="page-container">
  <header class="page-header">
    <h1>Mis quinielas</h1>
    <a routerLink="create" class="btn btn-primary">Crear nueva</a>
  </header>

  @if (loading()) {
    <p>Cargando...</p>
  } @else if (error()) {
    <p class="error">{{ error() }}</p>
  } @else if (pools().length === 0) {
    <div class="empty-state">
      <p>Aún no perteneces a ninguna quiniela.</p>
      <a routerLink="create" class="btn btn-primary">Crear tu primera quiniela</a>
    </div>
  } @else {
    @if (poolsByRole().asCreator.length > 0) {
      <section>
        <h2>Eres creador</h2>
        @for (pool of poolsByRole().asCreator; track pool.id) {
          <a [routerLink]="[pool.id]" class="pool-card">
            <h3>{{ pool.name }}</h3>
            <p>{{ pool.memberCount }} miembros</p>
          </a>
        }
      </section>
    }
    @if (poolsByRole().asMember.length > 0) {
      <section>
        <h2>Participas</h2>
        @for (pool of poolsByRole().asMember; track pool.id) {
          <a [routerLink]="[pool.id]" class="pool-card">
            <h3>{{ pool.name }}</h3>
          </a>
        }
      </section>
    }
  }
</div>
```

> **Nota:** se usa la nueva sintaxis de control flow (`@if`, `@for`) en lugar de `*ngIf` y `*ngFor`. Es la recomendación oficial en Angular 17+.

## Services con HttpClient

```ts
// src/app/features/pools/pools.service.ts
import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';

import { environment } from '../../../environments/environment';

export interface Pool {
  id: string;
  name: string;
  description: string;
  memberCount: number;
  role: 'creator' | 'admin' | 'member';
}

@Injectable({ providedIn: 'root' })
export class PoolsService {
  private http = inject(HttpClient);
  private baseUrl = `${environment.apiUrl}/api/v1/pools`;

  list(): Observable<Pool[]> {
    return this.http.get<{ items: Pool[] }>(this.baseUrl)
      .pipe(map(r => r.items));
  }

  getById(id: string): Observable<Pool> {
    return this.http.get<Pool>(`${this.baseUrl}/${id}`);
  }

  create(input: { name: string; description: string }): Observable<Pool> {
    return this.http.post<Pool>(this.baseUrl, input);
  }
}
```

## Formularios reactivos

```ts
// src/app/features/pools/pool-create/pool-create.component.ts
import { Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';

import { PoolsService } from '../pools.service';

@Component({
  selector: 'app-pool-create',
  standalone: true,
  imports: [ReactiveFormsModule],
  templateUrl: './pool-create.component.html',
})
export class PoolCreateComponent {
  private fb = inject(FormBuilder);
  private poolsService = inject(PoolsService);
  private router = inject(Router);

  submitting = signal(false);

  form = this.fb.nonNullable.group({
    name: ['', [Validators.required, Validators.minLength(3), Validators.maxLength(80)]],
    description: ['', [Validators.maxLength(500)]],
  });

  submit() {
    if (this.form.invalid || this.submitting()) return;
    this.submitting.set(true);
    this.poolsService.create(this.form.getRawValue()).subscribe({
      next: pool => this.router.navigate(['/pools', pool.id]),
      error: () => this.submitting.set(false),
    });
  }
}
```

## Guards funcionales

```ts
// src/app/core/auth/auth.guard.ts
import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from './auth.service';

export const authGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  if (!auth.isAuthenticated()) {
    router.navigate(['/login']);
    return false;
  }
  return true;
};
```

## Interceptors funcionales

```ts
// src/app/core/auth/auth.interceptor.ts
import { inject } from '@angular/core';
import { HttpInterceptorFn } from '@angular/common/http';
import { AuthService } from './auth.service';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const token = auth.token();
  if (!token) return next(req);
  const authReq = req.clone({
    setHeaders: { Authorization: `Bearer ${token}` },
  });
  return next(authReq);
};
```

## Componentes compartidos

```ts
// src/app/shared/components/team-flag/team-flag.component.ts
import { Component, input } from '@angular/core';

@Component({
  selector: 'app-team-flag',
  standalone: true,
  template: `
    <span class="team-flag" [attr.aria-label]="name()">
      <img [src]="flagUrl()" [alt]="''" [class.small]="small()" />
      @if (showName()) {
        <span class="team-name">{{ name() }}</span>
      }
    </span>
  `,
  styleUrls: ['./team-flag.component.scss'],
})
export class TeamFlagComponent {
  code = input.required<string>();
  name = input.required<string>();
  flagUrl = input.required<string>();
  small = input(false);
  showName = input(true);
}
```

> Uso del nuevo `input()` API (Angular 17.1+) en lugar de `@Input()`. Mejor type-safety.

## Estado global (cuando hace falta)

Para estado global de usuario, tema actual, etc., usar **Signals con services**:

```ts
// src/app/core/auth/auth.service.ts
import { Injectable, signal, computed } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private _user = signal<User | null>(null);

  readonly user = this._user.asReadonly();
  readonly isAuthenticated = computed(() => this._user() !== null);

  login(user: User, token: string) {
    localStorage.setItem('auth_token', token);
    this._user.set(user);
  }

  logout() {
    localStorage.removeItem('auth_token');
    this._user.set(null);
  }

  token(): string | null {
    return localStorage.getItem('auth_token');
  }
}
```

Para estado más complejo (cache de pools, predicciones), considerar `ngrx/signals` (no `ngrx/store` clásico — es demasiado para este proyecto).

## Convenciones

- Un componente = una carpeta con `.ts`, `.html`, `.scss`
- Templates inline solo para componentes triviales (< 10 líneas HTML)
- Servicios al final de la carpeta del feature (no en `core/` salvo singletons)
- Tipos compartidos en `*.types.ts` dentro de su feature
- Rutas en `*.routes.ts` por feature
- `inject()` en lugar de constructor injection (más limpio en standalone)

## Antipatrones

❌ **NgModules.** En Angular 17+ ya no se usan para nada salvo retrocompatibilidad.

❌ **`subscribe` sin manejar el unsubscribe.** Usar `takeUntilDestroyed()` o `async pipe`.

❌ **Lógica de negocio en componentes.** Va en services.

❌ **Importar `CommonModule` cuando solo se usa el nuevo control flow.** `@if`/`@for` no requieren importarlo.

❌ **`@Input()` decorator en código nuevo.** Usar `input()` signal-based.
