# SPEC-0058: исходная offline проверка реализации

Этот отчёт фиксирует состояние до production rollout. Текущий результат: deployed `db1d63d`, один product call, raw767/768 → одна UI-правка → candidate768/768; [production evidence](0058-production-evidence.md) содержит актуальные статусы, bill и оставшиеся ограничения.

Принятый контракт: [SPEC-0058](0058-real-document-import.md).
Принятие владельцем «Приступить к реализации этой спеки» зафиксировано
2026-10-26 в `cb4f100` до production-кода и интегрировано в локальный main.
Implementation source: `c2db56ce83ef31c4a65fd12b472477840c02c37b` (основной срез `e52f3cb`,
дополнен проверенной совместимостью pre-0058 upload replay и привязкой raw reply к текущей попытке),
ветка `codex/real-document-import`.

Состояние: **offline implementation verified / ready for review**.
Новый платный product run, live qualification фото/скана, integration в remote
main, production configuration/deployment и владелец-приёмщик не подменяются
локальными тестами. Старый research PR134 остаётся отдельным: его CLI не
используется в runtime и не требуется для интеграции этой реализации.

## Что реализовано

Отдельные `plain_document_reading` и `plain_review`: пять изображений без native
text/эталона/schema, исходный ответ текущей попытки отдельно от draft, серверный parser signed
minor units и дат/времени, редактирование/reload, единая сверка документа с
source/draft digest, существующий batch writer с preview/confirmation.
OpenAPI и сгенерированные TypeScript-типы обновлены. Новых финансовых writers нет.
Manual path работает с выключенным AI. Поштучные visual checks/quotes и strict
schema старых режимов сохраняются. Миграция 31 additive, retention удаляет
полный plain state, а downgrade с ним отклоняется.

## AC и исполняемые доказательства

| AC | Результат | Доказательство |
| --- | --- | --- |
| A1: вход | VERIFIED offline | `go test ./internal/documentdata`; `TestS56RealRenderNativeAndScan`, `TestS56RealRendererRefusesSixthPage`, `TestS58FivePageProductAttempt` через networkless helper. Пять страниц настоящего PDF прошли Web upload и доступны в browser; просмотрены 1–5. Полностью белый input/hash tamper отвергаются до POST |
| A2: actual wire | VERIFIED offline, provider fake | `go test ./internal/assistant -run TestPlainProductWireFivePagesWithoutEtalonOrSchema`: production constructor/DocumentReasoner, пять изображений, один POST/3 metadata GET, schema/native text/эталон/tools отсутствуют, reasoning явно disabled как в успешном prototype; нет structured-output requirements для plain, старый профиль требует их и максимум четыре изображения. SQL attempt/reserve/replay — `TestS58FivePageProductAttempt` |
| A3: parser | VERIFIED offline | `go test ./internal/statementdata -run TestPlain`; private saved-answer replay: 96 строк/768 checks PASS, итоги согласованы. Сверка — с независимой локальной расшифровкой текстовых колонок реального PDF; статус эталона остаётся draft. Негативы: переставленные колонки, обрезанная/лишняя строка, pipe, потеря времени без восстановления часов, неоднозначные суммы/валюты, forged typed values |
| A4: draft/UI | VERIFIED offline/browser | `StatementPlainDraft.test.tsx`, `StatementAI.test.tsx`: общий review, page viewing, stale edit, escaped raw text. В локальном Web реальный файл дал пять страниц; сохранённый ответ воспроизведён через **manual-drafts**, 96 строк показаны, правка пережила reload, единая сверка сохранена. SQL дополнительно подтверждает: manual edit не подменяет raw answer, а следующая явно согласованная попытка показывает свой ответ вместо старого. Это replay, не новая product inference |
| A5: финансовая запись | VERIFIED SQL/browser | `TestStatementPlainReviewConfirmAndInvalidation`: unreviewed/forged digest/incomplete pages/stale preview отвергаются, правка отменяет review, confirm replay не дублирует Posting, raw state исчезает при retention. В браузере public fixture прошёл review → решение purchase → preview → confirm → reload. В отдельной QA базе events=1/postings=1/publishedImportRows=1/aiUsage=0; реальные операции владельца не опубликованы |
| A6: один новый product call на реальном файле | **PENDING LIVE AUTHORIZATION** | Browser проверил загрузку/редактирование/review реального PDF с сохранённым ответом, а writer — на публичном synthetic документе. Новый платный вызов через продукт и независимое принятие эталона владельцем не выполнялись |
| A7: форматы | PARTIAL / live quality UNVERIFIED | Локальные renderer/normalizer/fake-provider контракты для PDF/photo/scan проходят. Реальные photo/scan quality examples по новому plain path отсутствуют; успех одного native PDF на них не переносится |
| A8: выпуск | RELEASE PENDING; CI отдельно | Локальные gates ниже PASS. Remote PR/CI отмечаются отдельно. Production flags, groups, budget allowance и service release не менялись; post-merge/deploy smoke ещё не выполнены |

## Найденная причина белого изображения

Новый настоящий пятистраничный PDF с невстроенным Helvetica воспроизвёл белые
страницы в прежнем helper: текст извлекался, но fallback fonts отсутствовали.
После `font-dejavu=2.37-r6` и временного XDG cache те же страницы стали читаемыми;
у runtime сохранены network=none, read-only root, uid10001, capabilities/seccomp,
memory/pid/tmpfs bounds. Версия пакета проверена в
[официальном Alpine v3.22 registry](https://pkgs.alpinelinux.org/package/v3.22/main/x86_64/font-dejavu).
Публичные cards/columns визуально просмотрены после исправления.

Старые corpus bytes и verdicts не переписаны. У двух исправленных полных cases
отдельно закреплены новые hashes в renderer integration test; остальные cases
и общий manifest проверяются прежней точной сверкой. Это не новый PASS закрытой
серии и не доказательство, что каждый её отказ объясняется шрифтами.

## Обязательные проверки

- `TMPDIR=/var/tmp/grace0058-tests make test` — PASS: Go, 341 Web/91 files,
  ESLint, FSD, Prettier, generated API/TypeScript/Vite, Jev и corpus checks.
- `make test-integration` — PASS: весь существующий набор disposable PostgreSQL
  и PDF-helper suites, включая отдельную базу для S58 после unknown-bill control.
- `./scripts/test-bank-statements.sh` — повторный PASS после последней правки
  detail projection: plain не получает native locator workflow.
- `go vet ./cmd/... ./internal/...`, `git diff --check` — PASS.
- `TMPDIR=/var/tmp/grace0058-tests make test-production` — PASS, только offline
  guards/production-bundle tests; deployment/live SSH/model calls=0.
- Pre-0058 upload idempotency: `TestStatementOfflineLifecycle` записывает прежнюю
  форму request hash и успешно воспроизводит её новым бинарником; SQL suite PASS.
- Private replay: `GRACE_PLAIN_SAVED_ANSWER=<private> GRACE_PLAIN_REFERENCE=<private>
  go test ./internal/statementdata -run '^TestPlainPrivateSavedAnswer$' -count=1 -v`
  — PASS, 96/768, provider calls=0. CI без этих env явно SKIP только private replay,
  обычные публичные parser/wire/SQL tests выполняются.

Обычный `make test` первоначально обнаружил посторонний read-only `/tmp/.git`:
старые privacy tests правильно отвергли output inside repository. Marker не
удалялся и guards не ослаблялись; для tests выбран отдельный внешний TMPDIR.
Первая новая SQL fixture также встретила ранее созданную одинаковую Posting:
matching корректно заблокировал импорт. Исправлены тестовые данные, не matching.

## Приватность и оставшиеся шаги

Существующий результат SPEC-0057 прочитан с прежнего VPS одним read-only SSH;
новых provider POST/metadata calls нет. PDF, исходный ответ, полный эталон,
временные UI/SQL данные и реквизиты не вошли в Git/PR/CI. QA использовала отдельный
Compose project с AI disabled, реальные финансовые записи production не менялись.

Готово к review исходников. Далее отдельно фиксируются remote CI и решение о
merge; live-приёмка в пределах предложенного в спеке лимита и production rollout
остаются явно неисполненными. Согласие на передачу и финансовое подтверждение
в приложении остаются действиями пользователя.

Локальный QA Compose project, его volumes с копией документа и Vite-процесс
остановлены/удалены после проверки. Исходный файл владельца не менялся.
