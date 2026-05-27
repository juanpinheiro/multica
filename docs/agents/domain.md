# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- **`CONTEXT-MAP.md`** at the repo root — it points at one `CONTEXT.md` per context. Read each one relevant to the topic.
- **`docs/adr/`** at the repo root — system-wide architectural decisions.
- **`<context>/docs/adr/`** — context-scoped decisions (e.g. `server/docs/adr/` for backend, `apps/web/docs/adr/` for web).

If any of these files don't exist, **proceed silently**. Don't flag their absence; don't suggest creating them upfront. The producer skill (`/grill-with-docs`) creates them lazily when terms or decisions actually get resolved.

## File structure

This is a multi-context monorepo. `CONTEXT-MAP.md` lives at the root and points to per-area `CONTEXT.md` files:

```
/
├── CONTEXT-MAP.md
├── docs/adr/                          ← system-wide decisions
├── server/
│   ├── CONTEXT.md
│   └── docs/adr/                      ← backend decisions
├── apps/
│   ├── web/
│   │   ├── CONTEXT.md
│   │   └── docs/adr/
│   ├── desktop/
│   │   ├── CONTEXT.md
│   │   └── docs/adr/
│   └── mobile/
│       ├── CONTEXT.md
│       └── docs/adr/
└── packages/
    ├── core/CONTEXT.md
    ├── views/CONTEXT.md
    └── ui/CONTEXT.md
```

When working on a cross-cutting change (e.g. a new API + UI), read every relevant `CONTEXT.md` — not just the one closest to the file you're editing.

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a test name), use the term as defined in the relevant `CONTEXT.md`. Don't drift to synonyms the glossary explicitly avoids.

If the concept you need isn't in the glossary yet, that's a signal — either you're inventing language the project doesn't use (reconsider) or there's a real gap (note it for `/grill-with-docs`).

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than silently overriding:

> _Contradicts ADR-0007 (event-sourced orders) — but worth reopening because…_
