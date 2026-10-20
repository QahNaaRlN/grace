# SPEC-0055: наблюдаемая сверка и исправление текущей разметки

- Дата: 2026-10-20, Asia/Yekaterinburg.
- Статус: **accepted**, 2026-10-20; прямое решение владельца «да, утверждаю».
- Implementation: **implemented_locally**, 2026-10-20; R1–R7 PASS локально,
  R8 **not_run** до отдельного release/deploy. Не merged/deployed.
- [Implementation evidence](0055-review-repair-evidence.md).
- Родитель: [SPEC-0055](0055-bank-statement-pdf-import.md), accepted technical
  contract и deployed S1.2. Конкретный контракт принят владельцем.
- Назначение: восстановить проверяемый ручной review без повторного AI-вызова.

## Основание и граница доказательств

Owner-test выявил расхождение сохранённых итогов заголовка при согласованных
суммах строк, неполное source coverage и пустой редактор исправления существующего
черновика. [Одобренная read-only проверка](https://github.com/QahNaaRlN/grace/actions/runs/39414784421)
подтвердила header mismatch и incomplete coverage без выгрузки текста/сумм
и без финансовой записи. Пустой редактор подтверждён отдельно UI и исходным кодом.
Узкая гипотеза пропущенного соседнего числового префикса **не подтверждена**.
Причина конкретного выбора исходных fragments остаётся неустановленной.

По коду `StatementManualDraft` начинает с `initial()`, а materialization сохраняет
Review без полного Proposal/service. Полный provenance для редактирования нельзя
восстановить дополнением к выбранным денежным fragments: остальные spans могли
быть как service, так и действительно unmapped. Ниже исправляется эта потеря
информации и её отображение, а не утверждается исправление качества модели.
Личные PDF, суммы, source UUID, имена и реальные locators в Git/CI не включаются.

## Результат для пользователя

При расхождении отдельно видны списания и зачисления: выбранный печатный итог,
сумма строк, статус, выбранные fragments и их контекст на странице. Если итог
не выбран или не распарсен, это явно показано вместо нуля. Переход к исходнику
подсвечивает selection; соседние слова доступны для самостоятельной проверки.
Отсутствующие book balances и доступный остаток сохраняют разные роли.

«Исправить группировку» открывает все текущие строки, header и сохранённые
service assignments. Изменение одного поля не удаляет остальные selection.
До сохранения видны изменения и предупреждение о повторной проверке черновика.
Save выполняет существующий manual-draft путь; не вызывает модель и не создаёт
финансовых событий. Отмена не меняет revision. Пустая новая форма разрешена только
для источника без текущего черновика, а не как замена существующего результата.

## Хранение и серверный контракт

1. Добавить nullable `mapping_snapshot jsonb` в `statement_sources`.
   Snapshot версии `statement-review-mapping.v1` содержит canonical Proposal,
   document layout hash, origin `model|owner`, связанный draft revision и
   точный список unmapped span IDs, вычисленный Go validator из фактических
   успешных claims (включая shared headings/currencies и service).
   Ошибочная selection не считается покрытием. Порядок service канонизируется
   существующим правилом; финансовые selection не исправляются и не расширяются.
2. Snapshot сохраняется атомарно с Review/records/revision в существующей
   `MaterializeStatementInTx` для AI и manual. Blocked, но reviewable Proposal
   тоже сохраняется; malformed документ/выход за лимиты остаётся rejected.
   UI не присылает authoritative coverage, money values или snapshot status.
3. Snapshot содержит locators/причины, не дублирует полный текст/страницы.
   Он приватен под существующей owner RLS; никакого raw model response,
   debug logging, публичных endpoints или расширения AI consent/transport.
   При удалении/истечении оригинала очищается вместе с layout, включая
   существующий DB trigger на original_deleted_at и все cleanup-пути. История audit
   хранит существующие безопасные metadata, без полной service разметки/текста.
4. Дополнить BankStatement в `api/openapi.yaml` optional `draftMapping`:
   `version`, `revision`, `status` (`complete|reconstructed_partial|unavailable`),
   nullable `proposal`, `unmappedSpanIds`, `unclassifiedSpanIds`, `issues`.
   `complete` означает сохранённость mapping, а не успешную сверку или право
   подтверждения. При exact snapshot `unclassifiedSpanIds=[]`, `unmappedSpanIds`
   содержит точный результат validator; при legacy точный unmapped список unknown
   и передаётся null, не выдуманный пустой список. API revision равен текущему
   Statement.Revision для expectedRevision. Snapshot отдельно помнит draft revision
   создания: последующее изменение retention/owner attestations само по себе
   не стирает mapping. Целостность snapshot проверяется по schema/layout hash;
   при несоответствии — unavailable с issue, без использования stale selections.
5. Legacy read-only восстановление: из Review.Evidence взять только сохранённые
   ordered IDs соответствующих row/header/label/currency ролей; направление взять
   из сохранённой validated Row.Direction, без вывода «расход/доход».
   Все текущие строки сохраняются в исходном порядке. Повтор одинакового shared
   header currency evidence сворачивается только при тождественном selection;
   конфликтующие/повреждённые evidence дают явный issue и поле без догадки.
   Service остаётся пустым и статус `reconstructed_partial`. Остальные layout IDs
   становятся **unclassified**, а не доказанно unmapped/service. Чтение ничего
   не записывает и не меняет финансовые guards. При отсутствии layout — unavailable.
6. Пользователь старого черновика может исправить header и явно распределить
   unclassified spans; submit проходит полную existing Go validation. Только
   сохранённый новый canonical Proposal становится complete snapshot, независимо
   от того, остались ли в нём видимые blockers. Автозаполнения service нет.
7. Сохранить current owner lock, expectedRevision, Idempotency-Key, published-row
   guard и транзакционность. Новый draft сбрасывает прежние record decisions,
   роли/проверенные страницы и инвалидирует previews, как текущая materialization;
   UI предупреждает об этом до save. После 409 несохранённые правки остаются
   видимыми; перезапись свежей server revision без нового review запрещена.
8. Money/checks остаются серверными integer minor units + currency. UI использует
   существующие Review.Header и Reconciliation debitMinor/creditMinor, не считает
   итоги. Сгенерировать TS types; вручную DTO в web не объявлять. Прежние limits
   Proposal/Document/retention сохраняются; не вводить скрытую усечённую выдачу IDs.

## Source coverage и интерфейс

Для полного snapshot показать точные unmapped spans с page/контекстом, отдельно
от owner page attestations и financial coverage. Для legacy показать предупреждение
«Служебная разметка старого черновика не сохранена» и unclassified fragments;
не утверждать, что каждый из них был пропущен моделью. Выбор fragment прокручивает
исходник. Показать количество и явную пагинацию/поиск при больших документах;
никакого молчаливого обрезания. Использовать mobile-first UI и существующий FSD.

Финансовое подтверждение по-прежнему блокируется при расхождении/неполноте,
неподтверждённых ролях/страницах/дубликатах. Модельные retries, исправление денег
по соседней геометрии и автоматическая смена policy в этот пакет не входят.

## Проверяемые критерии принятия

| ID | Критерий и исполняемое evidence после accepted |
| --- | --- |
| R1 | Go regression: synthetic split total с полным и неполным selection даёт matched/mismatch; отображаемые expected/computed значения и evidence точно соответствуют серверному Review, без изменения арифметики |
| R2 | Go/SQL: exact snapshot + computed unmapped IDs сохраняются для model/manual, включая blocked review; transaction rollback не оставляет snapshot/records; read round-trip сохраняет все selection/service reasons |
| R3 | Synthetic legacy fixture: все строки/header из evidence восстановлены; service не выдуман; unmapped=null, unclassified отделён; конфликтные IDs/отсутствующий layout явно обработаны, GET не повышает revision |
| R4 | Web: существующие несколько строк открываются целиком; правка одного header и отмена сохраняют остальные selection; new blank source имеет одну пустую строку; legacy предупреждение и exact/unclassified coverage различимы |
| R5 | HTTP/SQL: save revision/idempotency replay/conflict, owner isolation, published guard; preview invalidation/сброс старых решений; нет financialevents/postings/settlements до отдельного confirmation |
| R6 | SQL retention: delete/expiry удаляют snapshot и layout; остаётся только разрешённое selected financial evidence; snapshot не появляется у другого owner, в audit/logs/ошибках нет текста service |
| R7 | API generation + make test + make test-integration + git diff --check PASS; make test-production для сохранения activation/retention guards. Тесты используют synthetic fixtures, external calls=0 |
| R8 | После отдельного deploy повторный браузерный owner-test: selected totals и контекст доступны, текущие строки восстановлены, ручное исправление проверяемо; успешный импорт не объявлять без полного PASS и явного financial confirmation |

## Границы следующего решения

Утверждение этого контракта открывает реализацию R1–R7 и локальные проверки.
Публикация/интеграция/deploy выполняются в согласованном release процессе;
принятие контракта само по себе ничего не выпускает. Новые платные smoke/benchmark/личный inference,
изменения redaction/prompt/model/budget/feature flags не разрешаются этим пакетом.
S1.2 activation evidence сохраняется; owner E2E acceptance остаётся blocked.
