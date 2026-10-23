# PR №99 — интеграция принятой спецификации S1.17

Дата: 2026-10-23, Asia/Yekaterinburg. Основание: «влил PR».

- [PR №99](https://github.com/QahNaaRlN/grace/pull/99) MERGED,
  2026-10-23 12:33:11 +05:00 / 07:33:11 UTC;
  merge `94fd4abf4e2a97116d379474e4b692761ad061da`.
- Source head `97c1057d74b38c032914d705c0cb0002e3696ec2`:
  [CI39916489525](https://github.com/QahNaaRlN/grace/actions/runs/39916489525)
  SUCCESS5/5. Exact-source validator проверяет полный состав и успех всех jobs.
- [Merge CI39918524307](https://github.com/QahNaaRlN/grace/actions/runs/39918524307)
  SUCCESS5/5 на exact94fd4ab; проверен отдельно каноническим validator.
- [Delivery39919492659](https://github.com/QahNaaRlN/grace/actions/runs/39919492659)
  SUCCESS4/4 на том же merge: Verify source CI и три image publications.
  Это публикация образов; deployment не выполнялся.
- Primary main fast-forward81ef7e5→94fd4ab, status clean. Source head — предок
  merge, trees совпадают. Permissions primary0700 сохранены.

Принятый контракт [S1.17 E1–E5](0055-s117-alias-comparison.md) интегрирован.
На его основании создан новый worktree `statement-s117-comparison`, ветка
`codex/statement-s117-comparison`. [Отчёт реализации](0055-s117-evidence.md)
отделяет offline проверки от ещё не исполненного E6.

Локальные проверки новой реализации не называются post-merge тестом чистого94fd4ab.
Для merge здесь подтверждён полный exact-source CI; код в primary не редактировался.
Отдельное разрешение E6 получено ранее и не использовано. Оно требует приёмки
и интеграции реализации, закреплённого source/ref, его CI5/5 и preflight.
SSH/metadata/dispatch/inference в этом этапе не выполнялись.
