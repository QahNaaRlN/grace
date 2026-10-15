import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

import grace from './eslint-rules/grace.js'
import baseline from './eslint.baseline.js'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      globals: globals.browser,
    },
    rules: {
      'react-refresh/only-export-components': ['error', { allowConstantExport: true }],
    },
  },
  // SPEC-0048: architecture rules for the FSD layers.
  {
    files: ['src/**/*.{ts,tsx}'],
    ignores: ['src/shared/api/generated/**'],
    plugins: { grace },
    rules: {
      'grace/no-inline-query-key': 'error',
      'grace/no-direct-invalidate': 'error',
      'grace/no-time-zone-literal': 'error',
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@/shared/api/csrf',
              importNames: ['csrfToken'],
              message: 'The API client middleware sets X-CSRF-Token; do not pass it manually.',
            },
          ],
        },
      ],
    },
  },
  {
    files: ['src/entities/*/api/**'],
    rules: {
      'grace/no-inline-query-key': 'off',
      'grace/no-direct-invalidate': 'off',
    },
  },
  {
    files: ['src/features/*/api/**'],
    rules: { 'grace/no-direct-invalidate': 'off' },
  },
  {
    files: ['src/shared/config/**'],
    rules: { 'grace/no-time-zone-literal': 'off' },
  },
  {
    files: ['src/shared/api/**'],
    rules: { 'no-restricted-imports': 'off' },
  },
  ...baseline,
])
