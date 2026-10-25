# SPEC-0057: один простой запрос — ответ получен, сравнение совпало

Accepted [SPEC-0057](0057-real-document-simple-check.md), owner voice approval,
acceptance commit e3b30bec7f3fa881429d064927b42eb9a787076b **до кода**.
Исполненный source eaaa38faad2fc3b2cf2bbecae8253b7e3409327a из clean feature worktree,
Go1.24.0/CGO0/trimpath, explicit main.sourceSHA. Reviewable implementation: PR134.
Ни production deployment, ни включения профиля, ни финансового импорта не выполнялось.

| Критерий | Исполнение и executable evidence |
| --- | --- |
| Реальный файл/все страницы | Локально просмотрены все5 rendered PNG; каждый прошёл проверку hash/order/dimensions/ink в `real-document-check -mode=check`. Исходный PDF/страницы/эталон private, в GitHub не загружены. |
| Простой запрос | Один короткий русский prompt +5images/high в исходном порядке. Без response_format/JSON Schema/quotes/coordinates/native binding. Эталон и извлечённый текст не входят в input. `go test ./internal/assistant -run TestRealDocumentPlainWireAndLegacyBoundaries -count=1` PASS. |
| Старые product границы | Обычный NewModelGroups/старый NewExperimentModelGroups отклоняют5images; old provider сохраняет strict JSON и billing evidence при content refusal. Тот же causal test PASS; production composition не вызывает новый constructor. |
| Ограниченный transport/billing | Фактически1POST/3metadataGET, Gemini3.1FlashLite/Google Vertex EU, fresh proof/ZDR/no fallback/plugins/retries. Reserve35655micros, actual13771micros ($0.013771), BillingKnown=true/settlement=settled, input5583/output7415. При запуске no-clobber marker сохранён; второго запуска нет. |
| Guards/приватность | `go test ./cmd/real-document-check -count=1` PASS: blank/tamper/source/order/policy/no-golden/unknown-host/excess-call negatives. `python3 -B cmd/real-document-check/wrapper_test.py`5/5 PASS: marker/report/source/hash/symlink. Keys/DSN/operator identity не выводились. |
| SQL и regression | `make test` PASS (339 Web), `./scripts/test-ai-experiment.sh` PASS на одноразовом PostgreSQL, `go vet ./internal/assistant ./cmd/real-document-check` и `git diff --check` PASS. |
| Локальная сверка | Private `compare.py` разбирает полученную Markdown-таблицу, сравнивает каждую строку по8 полям без перестановки/дописанного времени/исправления значений; отдельно6 header/summary полей. Все ожидаемые строки/поля совпали, differences=[], банковские итоги совпали с суммами строк. Исходный модельный ответ сохранён без правок. |

Сравнение допускает только Unicode/пробелы, Markdown-экранирование, детерминированный
разбор signed decimal в minor units и символа валюты. На этом ответе даже formatting-only
differences отсутствуют. Эталон сохраняет draft_for_owner_review: это совпадение с нашей
предварительной расшифровкой одного реального файла, не общая qualification модели.

Личные реквизиты, операции, описания, телефоны, PDF/page hashes и ответ не публикуются.
Private result0600 сохранён на VPS; transient source-specific staging удалён wrapper cleanup,
container identity/image проверены unchanged. Локальные input/answer/reference/comparison
доступны владельцу в текущем чате. Никаких budget top-up, новых allowance или policy0 нет.
Закрытая прежняя синтетическая серия не возобновлялась, её verdicts/остатки slots не менялись.

Результат: **RESPONSE_RECEIVED / MATCHES_LOCAL_DRAFT_REFERENCE**.
Один разрешённый запрос CONSUMED; новые вызовы/автоматический import не открыты.
