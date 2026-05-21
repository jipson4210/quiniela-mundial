---
name: theme-system-3-palettes
description: How to implement the three-palette theme system (Mundial Vibrante, Dark Pro, Latina Cálida) for the Quiniela Mundial frontend. Use this skill whenever working on theming, color tokens, the theme switcher component, CSS custom properties, dark mode, or any visual styling that should respect the active palette. Apply this for any work under /frontend/src/styles/, when creating new components that need themed colors, or when the user mentions colors, themes, palettes, dark mode, or visual identity.
---

# Theme System — 3 Paletas

Sistema de temas intercambiables basado en **CSS Custom Properties** + persistencia en localStorage + service Angular.

## Las tres paletas

### 1. Mundial Vibrante (default)
Inspirada en los colores de las tres naciones anfitrionas (México, Canadá, EE.UU.) más el dorado del trofeo.

- Primary: verde mexicano `#006847`
- Secondary: rojo canadiense `#D52B1E`
- Accent: azul USA `#3C3B6E`
- Highlight: dorado trofeo `#FFD700`
- Background: blanco roto `#FAFAF7`
- Text: gris carbón `#1A1A1A`

### 2. Dark Pro
Modo oscuro profesional, ideal para uso nocturno y pantallas OLED.

- Primary: verde neón `#00E676`
- Secondary: ámbar `#FFB300`
- Accent: cian `#00B8D4`
- Highlight: blanco puro `#FFFFFF`
- Background: carbón `#121212`
- Surface: `#1E1E1E`
- Text: gris claro `#E0E0E0`

### 3. Latina Cálida
Inspirada en los colores de Manabí: tierra, ocre, cacao, océano. Identidad regional ecuatoriana.

- Primary: terracota `#C84B31`
- Secondary: ocre `#E8A33D`
- Accent: verde oliva `#6B8E23`
- Highlight: crema cacao `#F5DEB3`
- Background: arena `#FAF3E7`
- Text: chocolate `#3E2C1C`

## Estructura de archivos

```
frontend/src/styles/
├── _tokens.scss             # Variables semánticas (todas las paletas las definen)
├── themes/
│   ├── _vibrant.scss        # Mundial Vibrante (default)
│   ├── _dark-pro.scss       # Dark Pro
│   └── _latin-warm.scss     # Latina Cálida
├── _typography.scss
├── _spacing.scss
└── styles.scss              # Entry point: importa todo
```

## Tokens semánticos (`_tokens.scss`)

Definir variables **semánticas** (qué hace) en vez de literales (qué color es). Cada paleta luego las llena con sus propios valores.

```scss
// _tokens.scss
:root {
  /* Color tokens semánticos — cada theme los redefine */
  --color-primary: #006847;
  --color-primary-hover: color-mix(in srgb, var(--color-primary) 85%, black);
  --color-primary-contrast: #FFFFFF;

  --color-secondary: #D52B1E;
  --color-secondary-hover: color-mix(in srgb, var(--color-secondary) 85%, black);
  --color-secondary-contrast: #FFFFFF;

  --color-accent: #3C3B6E;
  --color-highlight: #FFD700;

  --color-success: #16A34A;
  --color-warning: #F59E0B;
  --color-danger: #DC2626;
  --color-info: #2563EB;

  --color-background: #FAFAF7;
  --color-surface: #FFFFFF;
  --color-surface-2: #F3F3EE;
  --color-text: #1A1A1A;
  --color-text-muted: #5C5C5C;
  --color-text-on-primary: var(--color-primary-contrast);

  --color-border: #E5E5E0;
  --color-divider: #EDEDE7;

  /* Sombras */
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px rgba(0, 0, 0, 0.07);
  --shadow-lg: 0 10px 25px rgba(0, 0, 0, 0.10);

  /* Radius */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-full: 9999px;

  /* Spacing scale (4-pt grid) */
  --space-1: 0.25rem;
  --space-2: 0.5rem;
  --space-3: 0.75rem;
  --space-4: 1rem;
  --space-6: 1.5rem;
  --space-8: 2rem;
  --space-12: 3rem;
  --space-16: 4rem;

  /* Typography */
  --font-family-base: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  --font-family-display: 'Manrope', 'Inter', sans-serif;
  --font-size-xs: 0.75rem;
  --font-size-sm: 0.875rem;
  --font-size-base: 1rem;
  --font-size-lg: 1.125rem;
  --font-size-xl: 1.5rem;
  --font-size-2xl: 2rem;
  --font-size-3xl: 2.5rem;
  --line-height-tight: 1.25;
  --line-height-base: 1.5;
  --line-height-relaxed: 1.75;
}
```

## Paletas

```scss
// themes/_vibrant.scss
[data-theme="vibrant"] {
  --color-primary: #006847;
  --color-primary-contrast: #FFFFFF;
  --color-secondary: #D52B1E;
  --color-secondary-contrast: #FFFFFF;
  --color-accent: #3C3B6E;
  --color-highlight: #FFD700;
  --color-background: #FAFAF7;
  --color-surface: #FFFFFF;
  --color-surface-2: #F3F3EE;
  --color-text: #1A1A1A;
  --color-text-muted: #5C5C5C;
  --color-border: #E5E5E0;
  --color-divider: #EDEDE7;
}
```

```scss
// themes/_dark-pro.scss
[data-theme="dark-pro"] {
  --color-primary: #00E676;
  --color-primary-contrast: #001F0E;
  --color-secondary: #FFB300;
  --color-secondary-contrast: #1F1500;
  --color-accent: #00B8D4;
  --color-highlight: #FFFFFF;
  --color-background: #121212;
  --color-surface: #1E1E1E;
  --color-surface-2: #2A2A2A;
  --color-text: #E0E0E0;
  --color-text-muted: #9E9E9E;
  --color-border: #333333;
  --color-divider: #2A2A2A;
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.30);
  --shadow-md: 0 4px 6px rgba(0, 0, 0, 0.40);
  --shadow-lg: 0 10px 25px rgba(0, 0, 0, 0.50);
}
```

```scss
// themes/_latin-warm.scss
[data-theme="latin-warm"] {
  --color-primary: #C84B31;
  --color-primary-contrast: #FAF3E7;
  --color-secondary: #E8A33D;
  --color-secondary-contrast: #3E2C1C;
  --color-accent: #6B8E23;
  --color-highlight: #F5DEB3;
  --color-background: #FAF3E7;
  --color-surface: #FFFCF5;
  --color-surface-2: #F0E5CB;
  --color-text: #3E2C1C;
  --color-text-muted: #7A6852;
  --color-border: #E0D5BD;
  --color-divider: #ECE2C9;
}
```

## Entry point

```scss
// styles.scss
@use 'tokens';
@use 'themes/vibrant';
@use 'themes/dark-pro';
@use 'themes/latin-warm';
@use 'typography';
@use 'spacing';

/* Global reset y base */
*, *::before, *::after { box-sizing: border-box; }
body {
  margin: 0;
  font-family: var(--font-family-base);
  background: var(--color-background);
  color: var(--color-text);
  line-height: var(--line-height-base);
  transition: background-color 0.2s ease, color 0.2s ease;
}

/* Componentes base reutilizables */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  font-weight: 600;
  border: 1px solid transparent;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-primary {
  background: var(--color-primary);
  color: var(--color-primary-contrast);
}
.btn-primary:hover {
  background: var(--color-primary-hover);
}

.btn-secondary {
  background: var(--color-secondary);
  color: var(--color-secondary-contrast);
}

.btn-outline {
  background: transparent;
  color: var(--color-primary);
  border-color: var(--color-primary);
}

.card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  box-shadow: var(--shadow-sm);
}
```

## Service Angular

```ts
// src/app/core/theme/theme.service.ts
import { Injectable, signal, effect } from '@angular/core';

export type ThemeName = 'vibrant' | 'dark-pro' | 'latin-warm';

const STORAGE_KEY = 'quiniela_theme';
const DEFAULT_THEME: ThemeName = 'vibrant';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  readonly current = signal<ThemeName>(this.loadFromStorage());

  readonly available: { id: ThemeName; label: string; description: string }[] = [
    { id: 'vibrant',    label: 'Mundial Vibrante', description: 'Colores oficiales de los anfitriones' },
    { id: 'dark-pro',   label: 'Dark Pro',         description: 'Modo oscuro profesional' },
    { id: 'latin-warm', label: 'Latina Cálida',    description: 'Inspirada en Manabí' },
  ];

  constructor() {
    effect(() => {
      const theme = this.current();
      document.documentElement.setAttribute('data-theme', theme);
      localStorage.setItem(STORAGE_KEY, theme);
    });
  }

  setTheme(theme: ThemeName) {
    this.current.set(theme);
  }

  private loadFromStorage(): ThemeName {
    const saved = localStorage.getItem(STORAGE_KEY) as ThemeName | null;
    if (saved && this.isValidTheme(saved)) return saved;

    // Auto-detectar dark mode si nunca se eligió
    if (window.matchMedia?.('(prefers-color-scheme: dark)').matches) {
      return 'dark-pro';
    }
    return DEFAULT_THEME;
  }

  private isValidTheme(t: string): t is ThemeName {
    return ['vibrant', 'dark-pro', 'latin-warm'].includes(t);
  }
}
```

## Theme switcher component

```ts
// src/app/shared/components/theme-switcher/theme-switcher.component.ts
import { Component, inject } from '@angular/core';
import { ThemeService } from '../../../core/theme/theme.service';

@Component({
  selector: 'app-theme-switcher',
  standalone: true,
  template: `
    <div class="theme-switcher" role="group" aria-label="Cambiar tema">
      @for (theme of themeService.available; track theme.id) {
        <button
          type="button"
          class="theme-option"
          [class.active]="themeService.current() === theme.id"
          (click)="themeService.setTheme(theme.id)"
          [attr.aria-pressed]="themeService.current() === theme.id"
          [title]="theme.description">
          <span class="theme-swatch" [attr.data-theme]="theme.id"></span>
          <span class="theme-label">{{ theme.label }}</span>
        </button>
      }
    </div>
  `,
  styleUrl: './theme-switcher.component.scss',
})
export class ThemeSwitcherComponent {
  themeService = inject(ThemeService);
}
```

```scss
// theme-switcher.component.scss
.theme-switcher {
  display: flex;
  gap: var(--space-2);
  padding: var(--space-2);
  background: var(--color-surface-2);
  border-radius: var(--radius-full);
}

.theme-option {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: none;
  background: transparent;
  border-radius: var(--radius-full);
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: var(--font-size-sm);
  transition: all 0.15s ease;

  &:hover { color: var(--color-text); }
  &.active {
    background: var(--color-surface);
    color: var(--color-primary);
    box-shadow: var(--shadow-sm);
  }
}

.theme-swatch {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 1px solid var(--color-border);
  flex-shrink: 0;

  &[data-theme="vibrant"]    { background: linear-gradient(135deg, #006847 50%, #D52B1E 50%); }
  &[data-theme="dark-pro"]   { background: linear-gradient(135deg, #121212 50%, #00E676 50%); }
  &[data-theme="latin-warm"] { background: linear-gradient(135deg, #C84B31 50%, #E8A33D 50%); }
}
```

## Reglas para usar los temas en componentes

✅ **Siempre usar variables CSS, nunca colores hardcoded:**

```scss
// ✅ Bien
.match-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  color: var(--color-text);
}

// ❌ Mal
.match-card {
  background: white;
  border: 1px solid #ddd;
  color: black;
}
```

✅ **Para colores semánticos de éxito/error/aviso**, usar las variables semánticas (`--color-success`, `--color-danger`, etc.) que son consistentes entre temas.

✅ **Para resaltar campos de pronóstico (acierto/fallo)**, derivar del primary:

```scss
.prediction-hit {
  background: color-mix(in srgb, var(--color-success) 15%, var(--color-surface));
  border-left: 4px solid var(--color-success);
}
.prediction-miss {
  background: color-mix(in srgb, var(--color-danger) 10%, var(--color-surface));
  border-left: 4px solid var(--color-danger);
}
```

## Accesibilidad

- Contraste WCAG AA mínimo (4.5:1) entre `--color-text` y `--color-background` en cada tema.
- `--color-text-on-primary` debe garantizar contraste contra `--color-primary`.
- Probar con [WebAIM contrast checker](https://webaim.org/resources/contrastchecker/) cada paleta.
- Respetar `prefers-reduced-motion`:

```scss
@media (prefers-reduced-motion: reduce) {
  * { transition: none !important; animation: none !important; }
}
```

## Antipatrones

❌ **Hardcodear colores** en componentes — siempre `var(--color-*)`.

❌ **Usar `:host-context([data-theme="dark"])` en cada componente.** Las variables CSS hacen la cascada automáticamente.

❌ **Crear variantes "dark" de cada componente.** Si las variables se usan bien, no hace falta.

❌ **Cargar JS de Tailwind/Bootstrap junto con este sistema.** Conflicto de tokens.
