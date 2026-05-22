import { Injectable, signal, computed } from '@angular/core';
import { HttpInterceptorFn, HttpRequest, HttpHandlerFn } from '@angular/common/http';
import { ApiService, User } from './api.service';
import { Router } from '@angular/router';
import { catchError } from 'rxjs';

const TOKEN_KEY = 'quiniela_token';
const USER_KEY = 'quiniela_user';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private _token = signal<string | null>(localStorage.getItem(TOKEN_KEY));
  private _user = signal<User | null>(this.loadUser());

  token = this._token.asReadonly();
  user = this._user.asReadonly();
  isLoggedIn = computed(() => !!this._token());

  constructor(private api: ApiService, private router: Router) {}

  private loadUser(): User | null {
    try {
      const raw = localStorage.getItem(USER_KEY);
      return raw ? JSON.parse(raw) : null;
    } catch { return null; }
  }

  register(email: string, password: string, displayName: string) {
    return this.api.register(email, password, displayName);
  }

  login(email: string, password: string) {
    return this.api.login(email, password);
  }

  saveSession(auth: { user: User; token: string }) {
    localStorage.setItem(TOKEN_KEY, auth.token);
    localStorage.setItem(USER_KEY, JSON.stringify(auth.user));
    this._token.set(auth.token);
    this._user.set(auth.user);
  }

  logout() {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
    this._token.set(null);
    this._user.set(null);
    this.router.navigate(['/login']);
  }

  getToken(): string | null {
    return this._token();
  }
}

export const authInterceptor: HttpInterceptorFn = (req: HttpRequest<unknown>, next: HttpHandlerFn) => {
  const token = localStorage.getItem(TOKEN_KEY);
  if (token && req.url.includes('/api/v1/') && !req.url.includes('/auth/')) {
    req = req.clone({ setHeaders: { Authorization: `Bearer ${token}` } });
  }
  return next(req);
};
