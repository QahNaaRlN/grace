# SPEC-0024: Implementation evidence

Accepted specification: [SPEC-0024](0024-interface-motion.md), acceptance
commit ff98699, direct owner request 2026-10-07.
Branch: `codex/interface-motion`. Implementation commit: 46b2b51.
Status: verified locally; main integration, remote CI and deployment pending.

## Delivered behavior

The favicon is a white heart/pulse on an indigo rounded square. HTML uses
`/favicon.svg?v=grace-1` to replace the previous browser-cached icon.
SVG has no external resources, scripts or font dependency.

Shared motion uses CSS tokens: enter 200ms, exit/interaction 140ms and
cubic-bezier(0.22,1,0.36,1). The side menu slides 8px from the right without
scaling its full-height drawer. Ordinary dialogs fade/translate/scale slightly;
overlays fade. Item-aligned selects fade without moving the selected option.
Buttons, nav links, rows and fields have short transitions; main initially fades
without route keys/remounts. No dependencies, handlers, timers or financial
contract changes were added. prefers-reduced-motion disables animations and
transitions; focus and disabled behavior remain available.

## Acceptance evidence

| Criterion | Executable evidence and result |
| --- | --- |
| AC1 | Headless Chromium loads SVG at 16/32/64px; all images decode with positive natural size. `bin/spec0024-favicon.png` visually inspected: heart/pulse remains readable. HTML versioned URL and SVG checked. PASS. |
| AC2 | Local `node bin/check-motion.cjs` with Playwright and synthetic API responses: computed menu animation `grace-menu-in`, 0.2s; closing `grace-menu-out`; ordinary form `grace-dialog-in`; item-aligned select `grace-fade-in`. Typed account name survives opening/closing its select. Motion CSS review confirms shared transitions and no route remount. PASS. |
| AC3 | Same browser check: Esc closes menu and select, focus returns to menu button. With reducedMotion=reduce, dialog/overlay animation-name=none, transition-duration=0s, and Esc/focus still work. Press scale excludes disabled/aria-disabled buttons; existing component disabled tests pass. PASS. |
| AC4 | 390×844 and 1280×844 browser checks: drawer right edge equals viewport edge, y=0, width=350, height=844; document has no horizontal overflow. Light/dark screenshots retained locally; mobile screenshot inspected. Browser pageErrors=[]; make test (29 Web tests, lint, build, Go and offline Jev checks) + go vet PASS; generated API has no diff; staged git diff --check PASS. |

## Reproduction and limits

Windows validation used Git Bash, a temporary ignored copy of the existing
Windows test adapter from the import worktree and an npm wrapper invoking
bundled Node 24.19.0:

`bash bin/test-windows-source.sh unit` runs `make test` and `go vet ./...`.
Browser smoke uses bundled Playwright, a local Vite server and only synthetic
read responses; no real financial requests were sent. Script/results and PNGs
are retained in ignored `bin/check-motion.cjs`, `bin/spec0024-browser-check.json`
and `bin/spec0024-{menu-390,menu-1280,menu-dark,favicon}.png`.

The sandbox-isolated local server was inaccessible to the browser; the local
preview was restarted outside network isolation and stopped after checking.
The production build has its existing large-bundle warning, but succeeds.
No new test/dependency or financial integration suite was introduced for this
presentation change. Browser checks ran in Chromium; other browsers and live
production release smoke remain release checks. Primary checkout remains clean.
