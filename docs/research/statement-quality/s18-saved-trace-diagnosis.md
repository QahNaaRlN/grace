# S1.8: границы установленного дефекта и следующий разбор

Дата: 2026-10-21, Asia/Yekaterinburg. Статический разбор main `5e9116e`.
Начальный раздел — статический срез до исполнения S1.9. Ниже добавлен
результат read-only диагностики, без новых model requests.

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

## Ограничения до исполнения S1.9

`scripts/research/diagnose-statement-smoke.py` поддерживает старые версии
statement.synthetic.v1/v2/v3 и выбирает qualification report через glob.
S1.8 имеет другой фиксированный путь и version statement.screening.v3.gemini31-identity.
`scripts/research/inspect-statement-screening-s18.py` проверяет aggregate/proof/ledger,
но не выводит field selections/причины; его workflow step привязан к paid job.
Запускать paid job повторно для доступа к inspector нельзя.

Полный private report остался на VPS и локально не копировался. До S1.9 его не
читали; прямой SSH ранее был недоступен, защищённый Actions SSH остаётся рабочим
операторским каналом. Нужен отдельный read-only path, не повтор smoke.

## Утверждённое доказательство

[Accepted S1.9](../../specs/0055-s19-saved-trace-diagnosis.md): получить только
закрытые reasons и locator selections фиксированного synthetic report; восстановить
документ pinned helper, сверить hashes и воспроизвести прежний Go validation/scorer.
До совпадения replay нельзя утверждать пропуск дефиса, неверный span, дефект
scorer или необходимость изменения prompt. Новый платный запуск и исправления
не входят в эту диагностику.

## Подтверждённая причина — 2026-10-21 03:48:32 +05:00

[Read-only workflow39510628212](https://github.com/QahNaaRlN/grace/actions/runs/39510628212)
SUCCESS, source `ecfa85078e0fae6d9f9e519f5e72b0126728fb45`.
Reader проверил fixed marker/source/profile/proof; artifactsUnchanged=true.
Replay на pinned helper совпал с сохранёнными checks19/correct17, codes,
validation и coverage. Полный report/ID/cost/owner не экспортированы.
[Закрытая проекция](s18-saved-trace-projection.json),
[вывод replay](s18-saved-trace-replay.json) содержат только public synthetic данные.

| Наблюдение | Точная причина |
| --- | --- |
| record_description | Строка2: вместо OWN TRANSFER - REVIEW SEMANTICS выбраны p2-b8-l1-w1,w2,w4,w5. Разделитель p2-b8-l1-w3 перенесён в service. Первая строка SYNTHETIC SHOP совпала |
| openingBookBalance:ambiguous_money | Выбран единственный `-` p1-b1-l1-w3 в заголовке SYNTHETIC BANK - COLUMNS; label взят из SYNTHETIC BANK. Начальный book balance в fixture отсутствует |
| closingBookBalance:ambiguous_money | Выбран единственный `-` p2-b1-l1-w3 в заголовке SYNTHETIC BANK - COLUMNS PAGE 2; label взят из SYNTHETIC BANK. Конечный book balance отсутствует |

Go не преобразовал эти два знака в деньги: normalization вернула ambiguous_money,
поля book balance остались nil, а оба blockers сохранились. Поэтому прошедший
absent_book_balances сам по себе не означает корректную исходную разметку:
неудачные попытки заполнить отсутствующие поля видны в trace и blockers.
Правильные суммы/обороты не позволяют игнорировать эти blockers.

Три standalone `-` были доставлены модели буквально и имели numeric_fragment;
они не скрывались под placeholders. Это подтверждено новым real-helper тестом
TestS19RetainedReplay, который проверяет также исходные selections и service.
ClassNumericFragment обозначает допустимую форму, а не финансовую роль.
Из этого нельзя заключать, что все самостоятельные знаки нужно исключить из
moneySelection: они нужны, например, для отдельно напечатанного знака суммы.

SystemPrompt v12 уже требует сохранять разделители описаний и оставлять
непечатные opening/closing balances пустыми. Следовательно, эти правила
существовали до вызова, но модель нарушила их в данном ответе. Новое добавление
похожей инструкции не доказано как исправление. Scorer/validation сработали;
это не основание ослаблять их или автоматически чинить модельный результат.

Следующее направление: закрепить этот counterexample в qualification corpus,
подготовить сравнение моделей/представления знаков при неизменных денежных
guards и scorer. Исправления prompt/schema/redaction/transport и любая новая
платная кампания требуют отдельного принятого контракта. S1.9 их не выполняет.
Cards/RU по этому кандидату всё ещё live UNVERIFIED, S1.8 остаётся FAIL.
