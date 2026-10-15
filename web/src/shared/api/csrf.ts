// CSRF token of the current session. Only the API client middleware reads it.
export let csrfToken = ''

export function setCSRFToken(value: string) {
  csrfToken = value
}
