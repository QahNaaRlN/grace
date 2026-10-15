// Baseline of architecture rule violations recorded at SPEC-0048 phase 0 (main e623775).
// Each later phase only removes entries; phase 5 deletes this file.
// Do not add new entries: fix the code instead.
export default [
  {
    files: [
      'src/features/create-account/ui/CreateAccountForm.tsx',
      'src/features/create-financial-event/ui/CreateFinancialEventForm.tsx',
      'src/features/create-obligation/ui/CreateObligationForm.tsx',
      'src/features/create-receipt/ui/CreateReceiptForm.tsx',
      'src/features/loan-details/LoanPanel.tsx',
      'src/features/obligation-schedule/ui/ObligationSchedule.tsx',
      'src/features/user-profile/ui/ProfilePanel.tsx',
    ],
    rules: { 'grace/no-direct-invalidate': 'off' },
  },
  // SPEC-0048 phase 3 moved these files from pages/home, pages/imports and pages/subscriptions.
  // Their direct invalidations become mutation hooks in phase 4 (owner decision 2026-10-15).
  {
    files: [
      'src/pages/imports/ui/ImportsPage.tsx',
      'src/pages/imports/ui/MonthPanel.tsx',
      'src/pages/subscriptions/ui/SubscriptionsPage.tsx',
      'src/widgets/accounts-list/ui/Accounts.tsx',
      'src/widgets/categories-list/ui/Categories.tsx',
      'src/widgets/ledger-screen/ui/LedgerScreen.tsx',
      'src/widgets/operations-list/ui/OperationEditor.tsx',
    ],
    rules: { 'grace/no-direct-invalidate': 'off' },
  },
  {
    files: ['src/pages/assistant/ui/AssistantPage.test.tsx'],
    rules: { 'grace/no-time-zone-literal': 'off' },
  },
]
