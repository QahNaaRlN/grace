# SPEC-0010 — implementation and verification evidence

- Specification: [0010-mobile-first-visual-system.md](0010-mobile-first-visual-system.md)
- Implementation commit: `2c135f4`
- Branch: `codex/mobile-first-ui`
- Verified: 2026-10-05

## Реализованный scope

- сохранена и закреплена компонентная система shadcn/ui `radix-nova`;
- обновлены общие color/radius/surface tokens, cards, inputs, selects, textarea и
  buttons;
- доступные controls получили pointer cursor, disabled controls — not-allowed;
- основные form controls имеют высоту 44 px, submit actions на телефоне занимают
  доступную ширину;
- одна semantic navigation адаптивно находится в header на desktop и fixed
  bottom safe-area на mobile;
- финансовый экран получил mobile-first иерархию: состояние, счета, аналитика,
  ввод и подробности;
- error-state теперь сообщает о частичной ошибке загрузки данных, не объявляя
  весь API недоступным;
- экран ассистента приведён к той же визуальной системе и mobile form flow.

## Acceptance criteria и исполняемые доказательства

1. Документация и библиотека: `docs/vision.md`, `docs/architecture.md`,
   `web/components.json`.
2. Cursor и touch targets: browser computed styles на 390 px подтвердили
   `cursor: pointer`; основные buttons/select triggers имеют высоту 44 px.
3. Mobile 390x844: `scrollWidth == clientWidth == 375`, один semantic `nav`,
   bottom navigation geometry `top=775`, `bottom=844`, console errors `[]`.
4. Assistant mobile: `scrollWidth == clientWidth == 375`, submit `303x44`,
   console errors `[]`.
5. Desktop 1440x1000: `scrollWidth == clientWidth == 1425`, navigation `static`,
   analytics отображается в двух колонках, console errors `[]`.
6. Финансовые ответы на smoke: status `Данные актуальны`, основные read endpoints
   загружены без ошибок.
7. API/OpenAPI/data changes: отсутствуют в implementation diff.

## Автоматические проверки

```text
make test
  go test ./cmd/... ./internal/...                         PASS
  vitest run                                               16 passed
  eslint .                                                 PASS
  openapi-typescript + tsc -b + vite build                 PASS
  jev dataset v1                                           49 validated
  jev dataset v2                                           72 validated

git diff --check                                           PASS
```

Vite сообщает не блокирующее предупреждение о JS chunk `645.13 kB`; ошибок
сборки нет. Оптимизация code splitting не входит в scope SPEC-0010.

## Visual smoke

- Mobile home: 390x844, данные загружены, no overflow, bottom navigation не
  перекрывает контент благодаря `pb-28` и safe-area inset.
- Mobile assistant: 390x844, privacy disclosure и primary action читаемы без
  zoom или horizontal scroll.
- Desktop home: 1440x1000, header navigation и two-column analytics работают.
- Browser console: no warnings or errors на обоих маршрутах и viewport.

## Итог

Все критерии SPEC-0010 закрыты исполняемыми и браузерными доказательствами.
Implementation готова к интеграции, но этот документ сам по себе не означает,
что commit уже достижим из `main`.
