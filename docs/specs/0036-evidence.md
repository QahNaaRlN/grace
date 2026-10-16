# SPEC-0036: реализация и проверка

Дата: 2026-10-13. Спецификация: [accepted SPEC-0036](0036-amount-input-profile-feedback.md).
Ветка: `codex/amount-input-profile`, база `f0fce1c` (принятые SPEC-0035–0037),
implementation commit `09fd14a`.
Статус: `integrated` — PR #22, merge `ed641ac`; post-merge `make test` в основной копии и CI `main` на `517e621` пройдены. Production не развёрнут (деплой вручную).

Обновление 2026-10-16 (сверка roadmap): release `8ad9705`, содержащий merge `ed641ac`, развёрнут 2026-10-14 (Deploy production run 38560119989).

| Критерий | Реализация | Исполняемое свидетельство |
|---|---|---|
| AC1 | `MoneyInput` (`shared/ui/money-input.tsx`) + `formatAmountText` (`shared/lib/amount-text.ts`): группировка NBSP при вводе, курсор сохраняется; валюта суффиксом и `aria-describedby` | `money-input.test.tsx` (7 тестов: форматирование, вставка `20 000 000,00 ₽`, FormData, курсор, controlled); `CreateObligationForm.test.tsx`: `20000000` → `20 000 000`, отправка `2000000000`; `HomePage.test.tsx`: `1 200,50` → `120050` |
| AC2 | Денежные поля обязательства, счёта, операции, подписки, кредита, черновика графика, чека, фото чека и импорта переведены на `MoneyInput`; разбор только `parseMinor`/`parseMilli` (BigInt) | `grep -rn 'type="number"\|valueAsNumber\|Math.round(.*\* *100' web/src` — только процент уверенности маршрута ассистента (не деньги); `CreateObligationForm.test.tsx`: три знака после запятой отклоняются, не округляются |
| AC3 | Статус в карточке действия: «Личные данные» / «Почта и вход» | `ProfilePanel.test.tsx`: `status.closest('section')` = «Личные данные», ошибка почты — в «Почта и вход» |
| AC4 | Кэш `auth-session` обновляется ответом `POST /api/auth/profile` (`user.name ← displayName`, `avatarUrl`) | `ProfilePanel.test.tsx`: двойной клик → `POST` один раз, `GET /api/auth/me` не вызывается, имя сессии обновлено |
| AC5 | `dirty` по trimmed значениям; после сохранения форма синхронизируется с ответом | `ProfilePanel.test.tsx`: кнопка неактивна до изменения и после сохранения |
| AC6 | Триггер «Добавить счёт» рендерится только на `/accounts` и при пустом списке на `/`, `/operations`, `/payments` | `HomePage.test.tsx`: на `/profile` при пустом списке кнопки нет |

Общие gates в worktree: `make test` — Go (все пакеты `ok`), web 21 файл / 79 тестов,
lint, build, офлайн Jev-проверки; `git diff --check` — чисто.

Решения по открытым вопросам: валюта — суффиксом в поле; импорт — сумма в рублях
(было «в копейках»). Фото чека отправляет сумму текстом без группировки
(`ungroupAmount`), потому что Go `receiptdrafts.ParseScaled` не принимает пробелы.

Не проверено: ручной просмотр на устройстве (iOS/Android клавиатура для
отрицательного остатка — `allowNegative` включает полную клавиатуру).
