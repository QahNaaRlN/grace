# SPEC-0055 / S1.8: один screening с идентичностью S1.7

- Дата: 2026-10-21, Asia/Yekaterinburg.
- Статус: draft; implementation_status: not_started; live_status: not_started.
- Acceptance record: отсутствует. Слияние PR №79 не разрешает новый inference.
- Основание: [S1.7](0055-s17-selected-model-identity.md),
  [I1–I4](0055-s17-evidence.md), [post-merge №79](0055-pr79-postmerge.md).
- Worktree: `statement-identity-smoke`; ветка `codex/statement-identity-smoke`,
  база `938b37062f6756c8f01106a26217752b5a4797ea` из main.

## Проблема и пользовательский результат

S1.7 проверен offline и включён в main. Его live качество ещё UNVERIFIED:
единственный S1.6 остановился до первого scorer, разрешение использовано.
Нужен отдельный ограниченный запуск с report v3, чтобы проверить прохождение
transport и прежних трёх форматов. Он не восстанавливает утраченный S1.6 envelope
и не доказывает историческую причину его отказа.

Результат: либо полный PASS columns19/19, cards20/20, RU23/23, либо честный FAIL
с безопасным классом selected model и фактическим состоянием учёта расходов.

## Scope и non-goals

Scope: отдельные one-shot runner/workflow mode, read-only preflight, один live
screening трёх прежних репозиторных синтетических PDF, private report/marker,
учёт по существующему ledger и проверка сохранности приложения до/после.

Не входят: повтор S1.4/S1.6, auto-retry/fallback, новые модели/маршруты,
изменение transport/пары имён/цен/бюджета/fees, prompt/planner/scorer/redaction,
расширение корпуса, личные PDF/drafts, финансовые записи, deployment,
переключение production policy/model/flags и activation по результату PASS.

## Точный профиль и денежные пределы

- Alias `google/gemini-3.1-flash-lite`; canonical только
  `google/gemini-3.1-flash-lite-20260507` по accepted S1.7; only route
  `google-vertex/eu`, display provider `Google`.
- До POST — свежие endpoint/capability/price/ZDR metadata и catalog proof S1.7.
  Любое несовпадение прекращает запуск без смены модели/маршрута/пределов.
- 100000 input / 16000 output tokens на документ; 120s на вызов, 420s на run.
  Ценовые ceilings — 275000/1650000 USD micros за миллион токенов.
- Один новый запуск: максимум **3 POST**, до **53900 micros на вызов**,
  общий provider cost cap **161700 micros ($0.1617)**. Прежний общий бюджет,
  allowance и fees сохраняются; их остаток не сбрасывается и не увеличивается.
- Whole-run reservation допускается только при достаточном остатке и отсутствии
  uncertain во всём shared ledger; проверяется существующими функциями.
- Порядок columns → cards → turnover-roles-ru. Первая ошибка завершает run;
  неиспользованные POST не переносятся, повтор без нового решения запрещён.

## Исполнение и инварианты

1. Новое фиксированное разрешение `s18-gemini31-identity-20261021`;
   private 0600 `/opt/grace/shared/statement-screening-s18-gemini31-identity-20261021.json`
   и `.started` с exact source SHA. Не читать и не перезаписывать старые
   artifacts/markers/audits S1.4/S1.6, не удалять consumed marker.
2. Новый отдельный workflow input `statement_screening_s18_only`, default=false;
   при нём другие screening/inspect/reconciliation/deploy jobs запрещены.
   `statement_action=preserve`, отсутствие reconcile modes обязательны.
   Любая конфликтующая комбинация не запускает платный/deploy job.
   Сохраняются production concurrency и существующий exclusive file lock.
3. Полный source SHA фиксируется после реализации. Перед оплатой должны пройти
   **все jobs CI exact SHA**, а также post-merge gates №79. Binary, SOURCE_SHA,
   runner и три PDF связаны manifest/SHA256SUMS; fixture hashes сверяются
   существующим screening CLI. Никаких личных документов и ключей в bundle.
4. Preflight без POST выполняется до расходования marker. Если он не прошёл,
   inference не разрешён и marker не расходуется. Наличие marker **или report**,
   в том числе symlink/orphan report, блокирует новый run до оплаты.
   Marker создаётся эксклюзивно непосредственно перед запуском; failure,
   interruption и другой source SHA не дают повторного разрешения.
5. Используется report `statement.screening.v3.gemini31-identity` и прежний
   validated RunScreening. Успешные calls содержат свежий на observedAt catalog
   proof, exact pin и закрытый selectedModelClass; timestamps согласованы с
   границами run. Отчёт без этих условий не является PASS.
6. Публичный вывод — только закрытые error/class/scorer codes, bool и counts;
   без raw model/provider/error/content/header/PDF, ключа, ID, owner и стоимости.
   Private correlation — unverified, не является known billing или разрешением
   auto-settlement. Временные binary/PDF/runner убираются; report/marker остаются.
7. Валидный billing учитывается существующим owner-scoped settlement, включая
   content FAIL. Неизвестный cost остаётся uncertain и блокирует новые платные
   запросы; никакого нуля по догадке, автоматического GET generation, ручного
   сброса ledger или использования старого S1.6 recovery для нового report.
8. После фактической попытки, в том числе failure, отдельно прочитать только
   фиксированный S1.8 private report и точную reservation. Inspector v3 читает
   aggregate/closed fields и shared uncertain flag, не ключи/личные данные;
   SQL только SELECT. Отсутствие/несовпадение report/row явно UNVERIFIED.
   Recorded calls=0 не трактуются как нулевой расход или отсутствие POST.
9. Проверить exact application container/image и health до/после. Config,
   flags, production model, финансовые данные и drafts не меняются. Private
   reports не выдаются как production qualification/включение новой модели.

## API, данные и rollback

Публичный API/OpenAPI/UI, миграции и финансовые модели не меняются.
Добавляется только operator authorization path и read-only inspector/mocks.
До smoke rollback — возврат operator/workflow коммита. После попытки marker,
report и ledger остаются честным audit: rollback не возвращает разрешение,
не отменяет фактический расход и не снимает uncertain по догадке.

## Критерии приёмки

| ID | Исполняемое подтверждение |
| --- | --- |
| S1 | Runner mocks: preflight failure POST0/no marker; failed/successful run расходует новый marker ровно раз; repeat/different SHA/orphan/symlink не делают POST; старые artifacts и hashes неизменны |
| S2 | Все сочетания реальных workflow job conditions взаимоисключены; новый mode default=false; exact source CI и bundle hashes проверяются до оплаты; inspect/reconcile/deploy/old smoke не исполняются совместно |
| S3 | Inspector v3: sanitization/sentinels, report version/pin/profile/proof validation, exact unique reservation/shared ledger только read-only; no private ID/cost/text в logs, old inspectors/recovery unchanged |
| S4 | make test, затронутые ops/integration/production mocks, go vet и git diff --check PASS; до отдельно утверждённого единственного run external inference=0 |
| S5 | Один live run после exact-source CI: полный PASS19/20/23 либо честный FAIL с diagnostics/metering и сохранёнными image/health; без повторов и переключения модели |

## Требуемое решение

Утвердить реализацию S1–S4 и один ограниченный live S5: максимум 3 вызова /
$0.1617 provider cap при прежнем общем бюджете/fees. Это новое разрешение,
потому что S1.6 использовано, а S1.7 явно исключало платный smoke.
До решения — draft/not_started; новые operator/workflow code и POST запрещены.
