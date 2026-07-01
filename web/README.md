# Sky Panel — web

The panel UI: React + TypeScript + Vite, TanStack Query, Zustand, Framer Motion, xterm.js for the live console.

## Development

```bash
npm install
npm run dev       # dev server, expects panel-api at http://localhost:8080 (override with VITE_API_URL)
npm run typecheck
npm run test
npm run build
```

## Structure

```
src/
  lib/          API client, auth store, theme engine, WebSocket topic hook
  components/   Shell (sidebar/topbar/animated background), console, admin tabs
  pages/        One file per route
  types/        DTOs mirroring panel-api's JSON responses
```

## Theming

The app ships with two presets (`Monochrome`, `Signal`) defined in `src/lib/theme.ts`.
Everything is driven by CSS variables (`--sp-*`) applied to `:root`, so the
in-app theme builder (`/account/theme`) can create and persist fully custom
themes to `localStorage` without touching any component code.
