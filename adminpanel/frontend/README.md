# Adminpanel Frontend (React + TypeScript + Vite)

Frontend админки TGApp.

## Quick start

```bash
cd adminpanel/frontend
npm i
npm run dev
```

## E2E smoke tests (Playwright)

Install browsers once:

```bash
npx playwright install chromium
```

Run smoke tests:

```bash
npm run test:e2e
```

Useful modes:

```bash
npm run test:e2e:headed
npm run test:e2e:ui
npm run test:e2e:report
```

## Support API integration

For live support chat in `Moderation -> Support`, set env vars before `npm run dev` or `npm run build`:

```bash
VITE_BACKEND_URL=http://localhost:8080
VITE_ADMIN_BOT_TOKEN=your_admin_bot_token
VITE_ADMIN_ACTOR_TG_ID=123456789
```

You can also set runtime fallbacks in `localStorage`:

- `backendUrl`
- `adminBotToken`
- `adminActorTgId`

## Admin login flow status

- UI экрана логина (`telegram -> 2fa -> password`) уже есть в `src/pages/LoginPage.tsx`.
- Полная интеграция с `adminpanel/backend/login` API пока не завершена (сейчас экран работает как UI flow).
- Готовый backend login API описан в `adminpanel/backend/login/README.md`.

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) uses [Babel](https://babeljs.io/) (or [oxc](https://oxc.rs) when used in [rolldown-vite](https://vite.dev/guide/rolldown)) for Fast Refresh
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) uses [SWC](https://swc.rs/) for Fast Refresh

## React Compiler

The React Compiler is not enabled on this template because of its impact on dev & build performances. To add it, see [this documentation](https://react.dev/learn/react-compiler/installation).

## Expanding the ESLint configuration

If you are developing a production application, we recommend updating the configuration to enable type-aware lint rules:

```js
export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      // Other configs...

      // Remove tseslint.configs.recommended and replace with this
      tseslint.configs.recommendedTypeChecked,
      // Alternatively, use this for stricter rules
      tseslint.configs.strictTypeChecked,
      // Optionally, add this for stylistic rules
      tseslint.configs.stylisticTypeChecked,

      // Other configs...
    ],
    languageOptions: {
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      // other options...
    },
  },
])
```

You can also install [eslint-plugin-react-x](https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-x) and [eslint-plugin-react-dom](https://github.com/Rel1cx/eslint-react/tree/main/packages/plugins/eslint-plugin-react-dom) for React-specific lint rules:

```js
// eslint.config.js
import reactX from 'eslint-plugin-react-x'
import reactDom from 'eslint-plugin-react-dom'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      // Other configs...
      // Enable lint rules for React
      reactX.configs['recommended-typescript'],
      // Enable lint rules for React DOM
      reactDom.configs.recommended,
    ],
    languageOptions: {
      parserOptions: {
        project: ['./tsconfig.node.json', './tsconfig.app.json'],
        tsconfigRootDir: import.meta.dirname,
      },
      // other options...
    },
  },
])
```
