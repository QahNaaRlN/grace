# S1.8: границы установленного дефекта и следующий разбор

Дата: 2026-10-21, Asia/Yekaterinburg. Статический разбор main `5e9116e`.
Новых model requests и чтения private report в этой сессии нет.

## Установлено

[S1.8 evidence](../../specs/0055-s18-evidence.md): columns checks19/correct17,
codes record_description/blockers, validation=[unknown,unknown], coverage=[].
В `internal/statementqualification/evaluate.go` Score проверяет описание каждой
из двух строк отдельно, и лишь одна такая проверка провалена: иначе было бы
два record_description и correct16 или ниже. Какой именно ряд — пока UNVERIFIED.
Ожидания scorer: строка1 SYNTHETIC SHOP, строка2 OWN TRANSFER - REVIEW SEMANTICS.
Это строки публичного synthetic fixture, не текст ответа модели.

Остальные 17 проверок прошли: row count, amount/currency, direction, две даты,
row_issues обеих строк, FX первой строки, turnover, coverage, absent book balances
и available balance; второе описание тоже прошло. Правильные числа не доказывают правильность field-label roles.
Два blockers пока нельзя расшифровать: публичный вывод закрыл их как unknown.
Scorer помещает Reconciliation.Blockers в Result.Validation; он не сохраняет
объяснения модели в этом поле. Прохождение row_issues сужает поиск, но не
доказывает конкретный header blocker или его связь с описанием.

Canonical route acceptance прошёл в S1.8; это не восстановление утраченного S1.6.
Billing известен/settled по независимому чтению 03:10:03 +05:00; unknown здесь
не означает unknown cost. Cards/RU не запускались и остаются live UNVERIFIED.

## Почему сейчас нельзя назвать причину

`scripts/research/diagnose-statement-smoke.py` поддерживает старые версии
statement.synthetic.v1/v2/v3 и выбирает qualification report через glob.
S1.8 имеет другой фиксированный путь и version statement.screening.v3.gemini31-identity.
`scripts/research/inspect-statement-screening-s18.py` проверяет aggregate/proof/ledger,
но не выводит field selections/причины; его workflow step привязан к paid job.
Запускать paid job повторно для доступа к inspector нельзя.

Полный private report остался на VPS, локально его нет. В этой сессии его не
читали; прямой SSH ранее был недоступен, защищённый Actions SSH остаётся рабочим
операторским каналом. Нужен отдельный read-only path, не повтор smoke.

## Следующее доказательство

[Draft S1.9](../../specs/0055-s19-saved-trace-diagnosis.md): получить только
закрытые reasons и locator selections фиксированного synthetic report; восстановить
документ pinned helper, сверить hashes и воспроизвести прежний Go validation/scorer.
До совпадения replay нельзя утверждать пропуск дефиса, неверный span, дефект
scorer или необходимость изменения prompt. Новый платный запуск и исправления
не входят в эту диагностику.
