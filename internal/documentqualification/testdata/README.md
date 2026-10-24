# Публичный набор SPEC-0056 Q

`corpus.json` создан `documentqualification.Prepare` через настоящий локальный
Poppler25.04.0 helper. Нормализация фото фиксирована на Go1.24.0
из go.mod; Prepare требует этот toolchain. Восемь cases, original/source/PNG/text/golden hashes,
настоящие native layouts только для native PDF. `Oracle` — публичный эталон
судьи; он не передаётся модели и не объявляется OCR фотографий/сканов.

`receipt-02.pdf` — image-only копия существующего synthetic receipt-02.png.
Генератор: `scripts/research/prepare-document-receipt-fixture.py`, ReportLab4.4.9
invariant. SHA256:4650a27fc30fdab5a159d7199e9ade5b8912b889382b08736b629ca5d493c137.
PDF реально отрендерен, текстовый слой отсутствует, PNG визуально проверен.

`candidate.json` содержит `null`: настоящая модель/endpoint не выбраны и не
квалифицированы. Default check не читает keys/env и не открывает сеть. Подмена
corpus/golden/prompt/schema после допуска меняет manifest и требует нового решения.

Проверить: `go run ./cmd/document-reading-eval -mode=check`;
`go test ./internal/documentqualification`;
`./scripts/test-document-reading.sh` — actual renderer parity и одноразовая SQL.
Регенерация corpus допускается только локально через `Prepare` и helper
с `GOTOOLCHAIN=go1.24.0`;
`TestQActualRendererCorpus` в integration-tag режиме сравнивает сохранённый
набор с настоящим renderer. Test-only `EXPORT_Q_CORPUS=/output/corpus.json`
сохраняет результат при изолированном writable output mount.

Ни одна из этих команд не доказывает точность настоящей LLM. Owner review,
financial readiness и production activation остаются отдельными проверками.
