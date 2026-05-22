import { Injectable, effect, signal } from '@angular/core';

export type ThemeName = 'mundial-vibrante' | 'dark-pro' | 'latina-calida';

interface ThemeOption {
  id: ThemeName;
  label: string;
  description: string;
}

const STORAGE_KEY = 'theme';
const DEFAULT_THEME: ThemeName = 'mundial-vibrante';
const ALL_THEMES: ThemeName[] = ['mundial-vibrante', 'dark-pro', 'latina-calida'];

@Injectable({ providedIn: 'root' })
export class ThemeService {
  readonly current = signal<ThemeName>(this.loadInitial());

  readonly available: readonly ThemeOption[] = [
    { id: 'mundial-vibrante', label: 'Mundial Vibrante', description: 'Verde, dorado y carmín de las sedes' },
    { id: 'dark-pro',         label: 'Dark Pro',         description: 'Modo oscuro para uso nocturno' },
    { id: 'latina-calida',    label: 'Latina Cálida',    description: 'Tierra, ocre y océano de Manabí' },
  ];

  constructor() {
    effect(() => {
      const theme = this.current();
      document.documentElement.setAttribute('data-theme', theme);
      try {
        localStorage.setItem(STORAGE_KEY, theme);
      } catch {
        // SSR or storage disabled — ignore
      }
    });
  }

  setTheme(theme: ThemeName) {
    if (this.isValid(theme)) this.current.set(theme);
  }

  private loadInitial(): ThemeName {
    let saved: string | null = null;
    try {
      saved = localStorage.getItem(STORAGE_KEY);
    } catch {
      // ignore — SSR or storage disabled
    }
    if (saved && this.isValid(saved)) return saved;

    if (typeof window !== 'undefined' && window.matchMedia?.('(prefers-color-scheme: dark)').matches) {
      return 'dark-pro';
    }
    return DEFAULT_THEME;
  }

  private isValid(t: string): t is ThemeName {
    return (ALL_THEMES as readonly string[]).includes(t);
  }
}
