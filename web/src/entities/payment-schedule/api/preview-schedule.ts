import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

export type SchedulePreviewRequest = components['schemas']['SchedulePreviewRequest']
export type SchedulePreview = components['schemas']['SchedulePreview']
export type RejectedScheduleLine = components['schemas']['RejectedScheduleLine']

export type PreviewResult =
  | { ok: true; preview: SchedulePreview }
  | { ok: false; message: string; rejectedLines: RejectedScheduleLine[] }

// SPEC-0037: Go builds, validates and totals schedules; the web only shows them.
export async function previewSchedule(body: SchedulePreviewRequest): Promise<PreviewResult> {
  const { data, error } = await apiClient.POST('/api/v1/obligations/schedule-preview', {
    body,
  })
  if (data) return { ok: true, preview: data }
  const failure = (error ?? {}) as { message?: string; rejectedLines?: RejectedScheduleLine[] }
  return {
    ok: false,
    message: failure.message ?? 'Не удалось построить график.',
    rejectedLines: failure.rejectedLines ?? [],
  }
}
