# Jev evaluation harness

`cmd/jev-eval` — development-only инструмент SPEC-0006. Он не подключён к
Grace API, не читает PostgreSQL и не выполняет assistant tools.

## Offline validation

Команда не требует сети или credentials:

```bash
go run ./cmd/jev-eval validate \
  --dataset testdata/jev-evaluation/v1/cases.json
```

Dataset содержит синтетические русские примеры и небольшую английскую
контрольную группу. Validator отклоняет неизвестные поля, дубликаты,
неподдерживаемые enum, несовместимые expected outcomes и известные классы
секретов/персональных идентификаторов.

## Live benchmark

Перед запуском API key задаётся только в окружении. В аргументах требуется
явный versioned model id; `jev-latest` и `jev-preview` запрещены, чтобы отчёт
был воспроизводимым.

```bash
TYPESAFE_API_KEY=<redacted> \
go run ./cmd/jev-eval run \
  --dataset testdata/jev-evaluation/v1/cases.json \
  --model jev-1.13.0 \
  --report /tmp/grace-jev-evaluation.json
```

Report path должен быть абсолютным и находиться вне репозитория. Новый файл
создаётся с mode `0600`; существующий файл не перезаписывается.

Evaluator отправляет три независимых `Choice` в одном запросе:

- read-only tool route, `clarify` или `unsupported`;
- форму периода без календарной арифметики;
- достаточность временных аргументов.

## Report

JSON report содержит:

- версии dataset и question set;
- requested и фактические model ids;
- token usage и latency по каждому case;
- ожидаемые и фактические typed answers с probabilities/confidence;
- отдельные метрики для `ru` и `en`;
- confusion matrix, accuracy, clarification rate;
- confidence buckets и coverage/error rate для порогов
  `0.50/0.60/0.70/0.80/0.90/0.95`;
- идентификаторы high-confidence ошибок;
- отдельные provider/protocol/composition errors.

Порог из отчёта не становится production policy автоматически. Live report
не добавляется в Git. Реальные пользовательские формулировки запрещены до
отдельного privacy/legal review.
