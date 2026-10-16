# SPEC-0038: реализация и проверка

Дата: 2026-10-13. Спецификация: [accepted SPEC-0038](0038-import-file-feedback.md),
GitHub issue [#14](https://github.com/QahNaaRlN/grace/issues/14).
Ветка: `codex/import-file-feedback`, база `06d055f` (SPEC-0035: `Notice`, `Field`),
implementation commit `590013c`.
Статус: `integrated` — PR #22, merge `ed641ac`; post-merge `make test` в основной копии и CI `main` на `517e621` пройдены. Production не развёрнут (деплой вручную).

Обновление 2026-10-16 (сверка roadmap): release `8ad9705`, содержащий merge `ed641ac`, развёрнут 2026-10-14 (Deploy production run 38560119989).

| Критерий | Реализация | Исполняемое свидетельство |
|---|---|---|
| AC1 | `pages/imports/NewDraftCard.tsx`: блок файла — имя, «Таблица CSV · 0,7 КБ», «Файл выбран — готов к сохранению», «Заменить файл», «Убрать» | `NewDraftCard.test.tsx` AC1 |
| AC2 | Проверка типа (`.csv`/`.json`), пустого файла и 16 MiB до отправки; ошибка через `Field` (`aria-invalid`) | `NewDraftCard.test.tsx` AC2: `.xlsx`, 16 MiB + 1 байт, запросов нет |
| AC3 | Состояние `saving`: «Сохраняем…», поле файла и кнопки недоступны, защита от повторного клика | `NewDraftCard.test.tsx` AC3/AC4: двойной клик → один `POST` |
| AC4 | Успех и ошибка — `Notice` внутри карточки; «Повторить» отправляет те же байты с тем же Idempotency-Key (ключ создаётся при выборе файла); `ImportsPage` больше не пишет сообщения формы вверху страницы | `NewDraftCard.test.tsx` AC3/AC4 и AC4 (повтор, одинаковый ключ); `ImportsPage.test.tsx` — исходные байты CSV и `sourceId` по-прежнему уходят в `/api/imports/csv` |
| AC5 | `role="status"` + `aria-live="polite"` для «выбран / сохраняем / сохранён»; `Notice` с иконкой и заголовком | `NewDraftCard.test.tsx` AC5/AC6 |
| AC6 | `input.tsx`: `disabled:cursor-not-allowed disabled:file:cursor-not-allowed` рядом с прежними `cursor-pointer` | `NewDraftCard.test.tsx` AC5/AC6 (классы); mock-превью: `cursor: pointer` у доступного поля |

Общие gates: `make test` — Go (все пакеты `ok`), web 26 файлов / 167 тестов,
lint, build, офлайн Jev; `git diff --check` — чисто.

#15 закрыт владельцем через Claude Code 2026-10-13 с комментарием о проверке
(отступы стрелки 13px/13px в mock-превью, значение не перекрывает стрелку).

Не проверено: выбор файла в настоящем браузере (встроенный браузер не
загружает файлы с диска) и screen reader вживую — поведение покрыто
тестами jsdom/Testing Library.
