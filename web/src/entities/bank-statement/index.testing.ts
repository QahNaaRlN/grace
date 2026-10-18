import type { components } from '@/shared/api/generated/schema'

export function statementFixture(): components['schemas']['BankStatement'] {
  return {
    id: 'statement-1',
    documentId: 'document-1',
    accountId: 'account-1',
    batchId: null,
    revision: 1,
    fileName: 'synthetic.pdf',
    sha256: '0'.repeat(64),
    status: 'ready',
    retention: 'delete_after_confirmation',
    expiresAt: '2026-10-26T00:00:00Z',
    deletedAt: null,
    createdAt: '2026-10-19T00:00:00Z',
    rolesConfirmed: false,
    reviewedPages: [],
    pageCount: 1,
    sizeBytes: 100,
    publishedRows: 0,
    excludedRows: 0,
    review: null,
    layout: {
      extractor: 'poppler-25.04.0/bbox-v1',
      pages: [{ number: 1, width: 600, height: 800 }],
      spans: [
        {
          id: 'date',
          page: 1,
          block: 1,
          line: 1,
          word: 1,
          box: { xMin: 1, yMin: 1, xMax: 50, yMax: 10 },
          text: '19.10.2026',
        },
      ],
    },
  }
}
export function statementPreviewFixture(): components['schemas']['StatementExtractionPreview'] {
  return {
    id: 'preview-1',
    statementId: 'statement-1',
    sourceRevision: 1,
    hash: 'preview-hash',
    expiresAt: '2026-10-19T00:15:00Z',
    plan: {
      documentHash: 'document-hash',
      promptVersion: 'statement-span-mapping.v1',
      schemaHash: 'schema-hash',
      redactionVersion: 'statement-allowlist.v1',
      policy: {
        model: 'google/gemini-2.5-flash',
        provider: 'google-vertex/eu',
        providerName: 'Google',
        inputMicrosPerMillion: 300000,
        outputMicrosPerMillion: 2500000,
      },
      maximumMicros: 8000,
      chunks: [
        { index: 1, context: [], spans: [{ id: 'date', page: 1, x: 1, y: 1, text: '19.10.2026' }] },
      ],
    },
  }
}
