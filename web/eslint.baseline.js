// Baseline of architecture rule violations recorded at SPEC-0048 phase 0 (main e623775).
// Each later phase only removes entries; phase 5 deletes this file.
// Do not add new entries: fix the code instead.
export default [
  {
    files: ['src/pages/assistant/ui/AssistantPage.test.tsx'],
    rules: { 'grace/no-time-zone-literal': 'off' },
  },
]
