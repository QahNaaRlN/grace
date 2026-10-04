# SPEC-0009 — verification evidence

- Состояние: `VERIFIED`
- Ветка: `codex/assistant-ui`
- Implementation commit: `37e35ba`
- Дата проверки: 2026-10-05

## Реализованный контракт

- добавлена отдельная Web-страница `/assistant` и общая навигация
  «Финансы»/«Ассистент» с `aria-current`;
- single-turn форма отправляет trimmed utterance, `language: "ru"` и текущий
  browser-local RFC 3339 timestamp с explicit offset;
- utterance ограничен `1..2000` UTF-8 bytes; blank и oversized input не
  вызывают API;
- один pending request блокирует повторный submit и отменяется при unmount;
- новый submit заменяет предыдущий результат, история не хранится;
- typed presentation реализована для accounts, obligations, receipts и period
  since latest income, включая явные empty states;
- `clarify`, `unsupported`, `unavailable`, validation, malformed и network
  errors имеют локальные безопасные тексты без raw response;
- privacy disclosure показывается до submit;
- страница не содержит write controls, не использует React Query cache,
  browser storage, URL state, console logging или client-side routing policy;
- OpenAPI explicit mappings исправили generated discriminator literals без
  изменения runtime wire contract;
- Vitest, Testing Library и jsdom добавлены как dev dependencies, `npm audit`
  при установке сообщил 0 vulnerabilities.

## OpenAPI evidence

После `make generate-api` generated contract содержит реальные значения:

```text
status: "completed";
status: "clarify";
status: "unsupported";
name: "get_accounts";
name: "get_obligations";
name: "get_receipts";
name: "get_period_since_income";
```

Поиск production-кода assistant UI не обнаружил `any`, `localStorage`,
`sessionStorage` или `console.*`. DTO импортируются из generated OpenAPI
contract через aliases в `web/src/shared/api/types.ts`.

## Interaction evidence

Команда:

```bash
cd web && npm test -- --run
```

Результат: 1 test file, 16 tests passed. Набор доказывает:

- отсутствие automatic request on mount;
- blank и UTF-8 oversized validation без API call;
- trim, язык `ru`, explicit-offset reference time и передачу `AbortSignal`;
- presentation всех четырёх completed result variants;
- clarify с required arguments, unsupported и оба unavailable codes;
- validation, malformed и network error presentation;
- один call при двойном submit и replacement предыдущего result;
- обычный Enter создаёт строку, `Ctrl+Enter` отправляет;
- unmount отменяет активный request.

## Visual evidence

Страница запущена через Vite и проверена в реальном браузере:

| Viewport | DOM evidence | Результат |
| --- | --- | --- |
| 1280 × 900 | `innerWidth=1280`, `scrollWidth=1280`; nav bounds `951..1136` | PASS: desktop layout и nav внутри viewport |
| 390 × 844 | `innerWidth=390`; элементов правее viewport нет; header/main width `375`, nav bounds `16..201` | PASS: нет horizontal overflow, nav переносится |

На mobile viewport textarea приняла «Покажи мои счета», `Ctrl+Enter` вызвал
submit, а недоступный локальный API отобразился как безопасный network state.
Viewport после проверки сброшен, временный Vite server остановлен.

## Pre-merge verification

| Проверка | Результат |
| --- | --- |
| `make test` | PASS: Go, 16 Web tests, ESLint, TypeScript/Vite, Jev offline |
| `make test-integration` | PASS: SPEC-0001..0005, 0007 и 0008 |
| `go vet ./cmd/... ./internal/...` | PASS |
| `git diff --check` | PASS |

Vite сохранил существующее предупреждение о chunk больше 500 kB; production
build успешен. Post-merge gates будут записаны после integration merge; до
этого SPEC-0009 имеет статус `verified`, а не `integrated`.
