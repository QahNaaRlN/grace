export function createIdempotencyKey() {
  return (
    globalThis.crypto?.randomUUID?.() ??
    `grace-${Date.now()}-${Math.random().toString(36).slice(2)}`
  )
}
