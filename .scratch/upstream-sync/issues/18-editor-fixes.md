# Issue 18: Editor fixes

**Status:** `done`
**Model:** `sonnet`

## Parent

PRD 1 — Upstream Sync (`.scratch/upstream-sync/PRD.md`).

## What to build

Two self-contained editor correctness fixes ported from upstream, grouped because they share the editor surface. Confirm each is absent locally before porting.

1. **Code-block render when auto-highlight is empty** — render fenced code blocks even when syntax auto-highlight returns an empty tree, so code stops vanishing.
2. **Preserve raw html-like text on paste** — pasting markup preserves the raw text instead of eating content.

## Acceptance criteria

- [x] A code block renders when auto-highlight returns an empty tree.
- [x] Pasting html-like text preserves it as raw text rather than dropping content.
- [x] Tests cover both editor behaviors.

## Blocked by

None — can start immediately.

## Comments

### Key decisions

- **Empty highlight fallback (`code-block-static.tsx` and `readonly-content.tsx`)**: When `toHtml(tree)` returns `""` (the hast tree from `lowlight.highlightAuto` has empty children), both rendering paths now fall back instead of producing an invisible code block. `CodeBlockStatic` uses `highlighted || escapeHtml(code)` in its `useMemo` so `dangerouslySetInnerHTML` always receives either the highlighted HTML or the safely-escaped raw text. `ReadonlyContent`'s `code` renderer branches on the empty string and returns `<code>{code}</code>` (React escapes children automatically), keeping the `dangerouslySetInnerHTML` path for the non-empty case only.

- **HTML-like paste (`markdown-paste.ts`)**: Added `looksLikeHtmlSnippet(text)` — a regex check for `/^\s*<[a-zA-Z!?/]/` — that detects text starting with an HTML/JSX/XML tag and routes it to the `"literal"` paste path instead of `"markdown"`. Without this, the `@tiptap/markdown` parser (markdown-it with `html: false`) silently strips HTML blocks, eating pasted markup entirely. The check is conservative: digits after `<` (e.g. `<3 pizza`) still go to the markdown path.

### Files changed

- `packages/views/editor/code-block-static.tsx` — added `|| escapeHtml(code)` fallback in `useMemo`
- `packages/views/editor/readonly-content.tsx` — split `toHtml` result into `highlighted` variable; branch on empty to use React-safe children fallback
- `packages/views/editor/extensions/markdown-paste.ts` — added `looksLikeHtmlSnippet` function and check in `classifyPaste`
- `packages/views/editor/code-block-static-fallback.test.tsx` (new) — 2 tests verifying the fallback renders content and escapes HTML when `toHtml` is mocked to return `""`
- `packages/views/editor/extensions/markdown-paste.test.ts` — 2 new tests: HTML-like text goes literal (parseSpy not called); `<3`-style text goes to Markdown

### Verification

- `pnpm --filter @multica/views test`: 83 files, 684 tests passed (up from 681)
- `pnpm typecheck`: 4 tasks successful, 0 errors
