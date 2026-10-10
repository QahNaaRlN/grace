# SPEC-0034: подписки, кредиты, документы, напоминания и OpenRouter

- Status: accepted
- Date: 2026-10-10
- Acceptance: прямое поручение владельца включить OpenRouter и провести проверки,
  добавить подписки, сведения/документы кредитов, копирование графика и почтовые
  напоминания за 3 и 1 день; уточнение: копия графика внутри Grace.
- Branch: codex/commitments-ai, base production d475cca.

## Scope and invariants

Подписка: название, сумма minor units/валюта, недельный/месячный/годовой период,
дата первого списания, необязательная дата окончания и карта/счёт. Пользователь
создаёт, редактирует, приостанавливает подписку; прогноз не создаёт проводку.
Фактические расходы связываются с подтверждёнными purchase; создание нового
расхода требует отдельного действия пользователя и использует финансовое ядро.
Связанный расход не считается повторно в аналитике. Календарные даты и месяцы
считаются Go в Asia/Yekaterinburg, anchor day сохраняется для конца месяца.

Кредит расширяет существующее Obligation(kind=loan): кредитор, номер договора,
необязательная выданная сумма и ставка, дата договора, график пользователя.
Сумма выдачи не является доходом; ставка не порождает автоматическую арифметику
или выдуманный банковский график. PDF-договоры сохраняются оригинальными bytes,
с SHA-256 и неизменяемыми версиями; доступны только владельцу через session API.
Размер ограничен 16 MiB/файл и 100 MiB/владелец, документы входят в DB backup.

Копия графика — редактируемый draft для того же обязательства. Копирование не
создаёт второе обязательство/расход. Применение требует отдельного подтверждения
и актуальной revision/base snapshot. Уже частично/полностью оплаченные строки
неизменяемы; выплаты и Settlements не удаляются. Старые неоплаченные планы
сохраняются неактивными. Конкурентная оплата делает старый draft устаревшим.

Напоминания: явный переключатель в форме подписки/кредита; только на текущую
verified email владельца. За 3/1 календарный день, после 09:00 Yekaterinburg.
Persistent outbox, unique source/due/lead, проверка актуальности перед отправкой,
общий сохранённый месячный лимит SMTP. Письмо содержит дату и ссылку Grace,
без финансовых сумм/названий. Перезапуск не повторяет подтверждённую отправку;
неопределённая SMTP-доставка помечается uncertain, без автоматического дубля.
Нет гарантии попадания во входящие; пропущенные окна не вызывают лавину писем.

RLS/owner composite FK, CSRF/Origin, exact money, idempotency и существующие
финансовые инварианты сохраняются. React отвечает за формы/представление.

OpenRouter: свежие model/provider/pricing/ZDR checks; одинаковый synthetic корпус
для 2–3 подходящих моделей, отказ при неизвестных обязательных данных. Метрика
для читаемых чеков >=95%, ошибки включены; никаких реальных данных без separate
in-app consent. Проверить существующий ключ/расходы и установить runtime allowance
в пределах оставшегося key limit и прежнего общего $10 с комиссиями, без сброса
usage. Production enable только с backup, non-root API/secret read check, exact
маршрутом и проверенным policy. Не обещать доступность после expiry policy.

Стоимость: актуальные официальные Yandex Cloud Postbox тарифы и фактические
доступные счётчики/биллинг; отличать измеренное использование, лимит и сценарий.
Не добавлять Yandex Compute/Storage автоматически: Grace уже размещён на VPS,
документы пока входят в PostgreSQL. Оценка хранения/backup оговаривает границы.

## Acceptance criteria

- AC1: subscriptions CRUD/pause, calendar projection and optional own-card binding;
  explicit actual purchase link/create; no forecast postings or duplicate totals.
- AC2: loan metadata and protected immutable PDF versions, quotas/integrity,
  cross-owner isolation; existing creation/payment behavior preserved.
- AC3: copy/edit/confirm schedule draft; paid lines/history preserved, stale
  concurrent settlement rejected, active-plan-only cash/payment queries.
- AC4: 3/1 day durable email reminders, verified-owner binding, restart/concurrency
  protection, cancellation and shared quota; failures visible without secret leaks.
- AC5: current OpenRouter checks/evaluation/budget evidence, consent and explicit
  write gates remain; non-root rollout/backup/readiness when privilege authorized.
- AC6: current Yandex cost report, typed OpenAPI regenerated, make test,
  integration gates, git diff --check and desktop/mobile browser evidence.
