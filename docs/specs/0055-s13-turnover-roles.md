# SPEC-0055 / S1.3: проверка ролей оборотов и актуальные кандидаты моделей

- [Интеграция PR №78 и отдельные post-merge проверки](0055-pr78-postmerge.md).
- Дата: 2026-10-20 (Asia/Yekaterinburg).
- Статус: accepted; implementation_status: integrated (PR №78, `29a28a7`, 2026-10-21); проверки (T1–T6 PASS; не deployed).
- [Отчёт и executable evidence](0055-s13-evidence.md).
- Acceptance record: прямое поручение владельца «оформи спеку и вперед» после
  принятого плана регрессии БИК/КПП и подбора современных эффективных моделей.
- Ветка: codex/statement-quality-next; baseline d538b78.
- Основание: [план](../research/statement-quality/next-quality-package.md),
  [диагноз/R8](0055-review-repair-evidence.md).

## Цель и ограниченный scope

Не принимать банковские реквизиты за обороты даже при совпадении чисел;
сохранить финансовые подписи в redacted request и подготовить проверенный
shortlist современных model/endpoint пар. Направление и реализация этого пакета
приняты прямым поручением; SPEC-0056 registry остаётся draft. Платные inference,
смена production policy, deployment и финансовое подтверждение сюда не входят.

## Контракт реализации

1. Только фиксированные общие подписи «Расходы», «Пополнения», «БИК», «КПП»,
   «ИНН», слова «по/дебету/кредиту» для поддержанных подписей и BIC/KPP/INN/SWIFT добавляются в allowlist. Значения реквизитов и
   неизвестные слова остаются скрытыми по прежним правилам, explicit visibility
   не раскрывает целые numeric identifiers. Prompt поясняет положительные и
   отрицательные роли, geometry и unknown; никакого дополнения ответа за модель.
2. PromptVersion v12.turnover-roles; RedactionVersion v4.financial-labels.
   Planner/aliases/schema/transport/лимиты/цены/бюджет/fees неизменны. Existing
   hashed-plan/profile guards отвергают старое согласие; нужен новый preview.
3. Чистый Go проверяет выбранные debit/credit turnover evidence, включая ручной
   draft. Поддерживаемые подписи после lower/punctuation/whitespace normalization:
   debit: Debit turnover, Расходы, Списания, Оборот(ы) по дебету;
   credit: Credit turnover, Пополнения, Зачисления, Оборот(ы) по кредиту.
   Неизвестная/смешанная/обратная подпись — blocker turnover_role_unverified.
   Пустой оборот остаётся unknown/not_provided, сумма из строк не подставляется.
4. Подпись занимает одну печатную строку; значение на той же странице, в той же
   строке справа либо следующей строке в пределах 24 pt с левым краем внутри
   ширины подписи. Несовместимая geometry — blocker turnover_role_unverified.
   Значение на строке рядом справа (до 100 pt) от явного БИК/КПП/ИНН/BIC/KPP/INN/
   SWIFT также блокируется, даже при чужой корректной подписи. Это консервативная
   проверка известных ролей, не универсальный парсер банков. Трудные layouts
   остаются reviewable с blocker, без автоматического выбора замены.
5. Выбранные значения, spans и суммы сохраняются для диагностики. Неверная роль
   переводит turnoverCheck в incomplete, добавляет blocker. Сравнение денег и
   строки не изменяются. Тот же guard применяется при чтении/preview старого
   сохранённого review, используя retained evidence и полный layout если доступен;
   не изменяет БД и не требует повторного платного вызова. Web показывает понятное
   пояснение нового blocker через существующий словарь issues, без новой бизнес-логики. После удаления layout
   остаются проверки подписи и geometry по retained evidence; это не обещание
   обнаружения утраченного невыбранного контекста.
6. Новый публичный двухстраничный RU synthetic fixture, независимый source/golden
   и реальные Poppler spans: обе роли, разделённые тысячи, реквизиты на footer,
   отрицательные ответы с неправильной подписью/значением, совпадением чисел,
   заимствованной подписью другой страницы и unknown. Не копировать private PDF.
   Existing columns/cards, descriptions и coverage не ослабляются.
7. Read-only public metadata baseline + до двух современных альтернатив:
   exact model/endpoint, timestamp, цены, capabilities, reasoning/profile,
   privacy verification status и eligibility. Не считать каталог доказательством
   ZDR или качества; не расширять allowlist автоматически. Сравнение качества/
   стоимости/latency на одинаковом corpus оформляется следующим bounded run.

## Критерии приёмки

| ID | Исполняемое evidence |
| --- | --- |
| T1 | Реальный RU PDF → Poppler layout → independent golden: правильные обороты matched, строки и coverage полны |
| T2 | Wrong-label, same-number identifier, borrowed-label, reversed-label, missing/unknown-role ответы не проходят; ссылки/значения не ремонтируются |
| T3 | Redaction сохраняет только новые фиксированные подписи; identifier values скрыты; старые prompt/redaction plans invalid |
| T4 | Retained legacy review/read/preview применяют guard, owner flags не снимают role blocker; manual остаётся доступным |
| T5 | Timestamped model/endpoint snapshot и shortlist, теоретические ceilings рассчитаны, privacy/quality UNVERIFIED отделены от допуска |
| T6 | make test, affected real-helper/SQL suite, go vet, git diff --check; provider calls = 0 |

Дальше: отдельно согласованный smoke выбранной пары на columns/cards + новом
fixture, с whole-run reserve и точным aggregate cap, затем expanded corpus.
S1.2 smoke уже использован. До новой qualification старая production policy
не меняется, AI-quality improvement остаётся UNVERIFIED.
