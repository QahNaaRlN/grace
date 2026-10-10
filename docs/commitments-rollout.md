# SPEC-0034: порядок выпуска

Спецификация [accepted](specs/0034-commitments-ai-reminders.md),
[проверки и стоимость](specs/0034-evidence.md). Этот документ не является
разрешением на новый privileged доступ. Предыдущее согласие root было ограничено
прошлым деплоем; для нового требуется разрешение владельца.

1. Проверить Linux CI для exact SHA: make test, test-production, integration,
   API/Web container build. Подготовить review/preview владельцу.
2. Проверить текущий release/auth mode, чистую миграцию 17, owner/identity binding,
   financial fingerprints и свободное место. Снять pre-release backup и
   проверить restore drill. Не переносить реальные документы в LLM.
3. Проверить существующий OpenRouter key из UID 10001, свежий key usage/remaining
   и endpoint prices. Установить защищённый `ai.env` по example с актуальным
   VerifiedAt и `GRACE_AI_ENABLED=true`; ключ не копировать в release.env/Git.
4. В транзакции с блокировкой `ai_experiment_budget FOR UPDATE` сверить все
   reserved/uncertain. При uncertain остановить выпуск AI до сверки invoice.
   Сохранить existing ai_usage и fees. Allowance = уже учтённый committed
   расход production + разрешённый новый остаток (не более 4 900 000 microUSD
   по снимку 2026-10-10 и не более свежего key remaining). Не сбрасывать usage.
5. Выпустить миграцию 18 и контейнеры через deploy.sh от grace-deploy. Protected
   ai.env автоматически включает compose.ai.yaml; init назначает права только
   photos volume, API остаётся UID 10001. Проверить readiness, публичный экран
   входа, авторизацию API, key read/photo write и protected capabilities.
6. Повторно сверить identity и financial fingerprints. До пользовательских
   записей новые таблицы должны быть пусты; суммы/история прежних данных не меняются.
   Проверить процесс reminder worker, настройки SMTP/budget и текущий счётчик
   auth_email_rate_windows для месяца. Reservations — верхняя оценка отправлений,
   а не полный cloud billing. Сравнить с Postbox metrics/bill, если доступны.
7. Проверка реальных данных проходит через пользовательский consent и draft
   confirmation; тесты/оператор не отправляют реальные фото или вопросы. Письма
   реальным владельцам не отправлять вручную для smoke без отдельного поручения.

Откат приложения: после неактивных строк старое приложение до миграции 18
может повторно учитывать историю. Guard откажет в таком откате; подготовить
совместимый исправленный SHA. Не делать DOWN с пользовательскими данными и
не восстанавливать backup поверх новых оплат автоматически.

Отдельное отключение AI: `GRACE_AI_ENABLED=false` в protected ai.env и повторный
non-root rollout. Конфигурация и история usage/consent не удаляются. Основные
подписки, кредиты и арифметика работают без LLM.
