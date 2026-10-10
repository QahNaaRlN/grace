# SPEC-0033: понятные счета, категории, финансовое здоровье и ИИ

- Status: accepted
- Date: 2026-10-10
- Acceptance record: прямое поручение владельца о картах, пользовательских
  категориях/выборках, скелетонах, OpenRouter, финансовом здоровье, file input
  и плавных аккордеонах. Проверка владельцем предшествует деплою.
- Branch: codex/ux-refinement; base codex/ui-redesign + accepted SPEC-0026.

## Scope

Карты показывать как «Банк *4517», не подменять отсутствующие реквизиты;
разные доступные иконки debit/credit/cash и подписи типов. Имя самого счёта
сохраняется в данных и для счетов без карты. Пользователь создаёт и переименовывает
свои категории расходов; общие категории read-only. Никакого удаления используемых
категорий/автоматической переклассификации. Категория выбирается при покупке,
выборка операций идёт на сервере до limit, с фильтром дат и категории.

Объединить готовые SPEC-0026 и UI/password/statistics локально. Ассистент:
LLM-анализ, явное согласие, доступность с сервера, понятные возможности и
ограничения; прежние детерминированные инструменты остаются доступны.
Гайд ИИ описывает реализованное отдельно от будущих возможностей.

Финансовое здоровье: понятные действия и качество данных вместо копии
spreadsheet с пустыми полями; достоверные расчёты на существующем Go API,
без выдуманного рейтинга, новых прогнозов или арифметики в React.
Единые skeleton/loading/error/empty состояния, file input padding,
Radix disclosure с ChevronDown, плавное изменение высоты и reduced motion.

## Инварианты

RLS/owner binding, CSRF/Origin, integer money, unknown=null, explicit AI write
confirmation по SPEC-0026 сохраняются. No production deployment until review.
Не использовать реальные финансовые данные/платный inference для визуальной QA.
Консолидировать номера ещё не выпущенных миграций без изменения production DB.

## AC

- AC1: единые bank/last4 названия и разные debit/credit icons во всех списках/формах.
- AC2: создание/rename personal expense categories, read-only globals, validation,
  cross-owner isolation, purchase selection, server-filtered period/category list.
- AC3: skeletons, file spacing, плавные доступные раскрытия и reduced motion.
- AC4: актуальный AI экран/гайд, серверная доступность и consent; понятный health
  screen со ссылками на расчёты и ограничениями полноты.
- AC5: make test, git diff --check, PostgreSQL integration для новых mutations и
  совместных миграций/auth/AI, browser desktop/mobile both themes и evidence report.
