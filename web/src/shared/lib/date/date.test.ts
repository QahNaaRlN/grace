import { expect, it } from 'vitest'
import { eventDateText, eventTimeText } from '.'

it('renders calendar history with unknown time rather than epoch or fabricated hours', () => {
  const event = { occurredAt: null, occurredDate: '2026-09-02' }
  expect(eventDateText(event)).toContain('время неизвестно')
  expect(eventTimeText(event)).toContain('2026')
  expect(eventTimeText(event)).not.toMatch(/00:00|12:00|1970/)
  expect(eventDateText({ occurredAt: null })).toBe('Дата неизвестна')
})
