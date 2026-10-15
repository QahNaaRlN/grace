import { defineConfig } from 'steiger'
import fsd from '@feature-sliced/steiger-plugin'

// SPEC-0048: every Feature-Sliced Design rule is an error; there is no baseline.
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
