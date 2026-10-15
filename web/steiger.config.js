import { defineConfig } from 'steiger'
import fsd from '@feature-sliced/steiger-plugin'

// Existing violations are matched by scripts/lint-fsd.mjs against steiger.baseline.json.
export default defineConfig([
  ...fsd.configs.recommended,
  {
    ignores: ['./src/shared/api/generated/**'],
  },
  {
    // SPEC-0048: a feature is a user action, so a slice used by one page is valid.
    files: ['./src/**'],
    rules: { 'fsd/insignificant-slice': 'off' },
  },
])
