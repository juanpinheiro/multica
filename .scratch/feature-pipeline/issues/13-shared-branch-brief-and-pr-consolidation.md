# Issue 13: Shared-branch brief — pular in_review, PR consolidado, listar sibling issues

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

Quando uma issue carrega `IsSharedBranch=true` (ou seja, herda o branch via `feature.target_branch` per Issue 02), o brief gerado pelo daemon (`server/internal/daemon/execenv/runtime_config.go::buildMetaSkillContent`) precisa instruir o agent a:

1. **Marcar a issue como `done` direto** ao terminar, NÃO `in_review`. O dependency gate (Issue 04) checa estritamente `b.status != 'done'`; deixar em `in_review` trava todas as dependentes indefinidamente, contradizendo o PRD User Story #15 ("1 feature = 1 PR, revisado uma vez no final"). Revisão humana acontece no PR consolidado depois que todas as sibling issues completam, não issue-por-issue.
2. **Não abrir um novo PR** quando o PR no branch shared já existe. O primeiro agent a fazer push abre o PR; agents subsequentes só fazem `git push` no mesmo branch — GitHub acumula os commits no PR existente automaticamente. Cada agent adiciona um comentário no PR (`gh pr comment <num> --body "<identifier>: ..."`) pra o reviewer conseguir mapear commits → issues.
3. **Usar título de PR feature-level**, não issue-level. Em vez de `feat(TES-1): add /integrations/telegram route`, usar `feat: Conectar Telegram` (derivado de `feature.title`). Description com seção `## Implements` listando TODAS as sibling issues por identifier.
4. **Listar siblings via CLI**: o agent pode rodar `multica issue list --feature <FeatureID> --output json` pra descobrir as outras issues da feature e popular o `## Implements`.
5. **Manter as regras de segurança existentes** (sem force-push, sem rewrite, pull --rebase antes de push) — esse bloco já existia na seção `## Shared branch` e não muda.

A mudança no workflow é condicional: o ramo `if ctx.IsSharedBranch` ativa o novo comportamento. Issues isoladas (sem `feature.target_branch`) continuam com o fluxo `in_review` como antes.

## Acceptance criteria

- [x] Seção `## Shared branch` do brief adiciona um sub-bloco `### Consolidated PR model` com instruções sobre PR title feature-level, `## Implements`, e regra "não abrir PR novo se existe".
- [x] Seção `## Shared branch` do brief adiciona um sub-bloco `### Status workflow under shared branch` explicando por que `in_review` é pulado e quando ainda usar `blocked`.
- [x] No `### Workflow` da assignment task, o passo 8 muda condicionalmente: `IsSharedBranch=true` → `multica issue status <id> done`; `IsSharedBranch=false` → `... in_review` (comportamento legado preservado).
- [x] Quando `FeatureID` está disponível, o brief inclui o comando concreto `multica issue list --feature <FeatureID> --output json` pra o agent descobrir siblings.
- [x] Quando `FeatureTitle` está disponível, o brief sugere o título do PR como `feat: <FeatureTitle>`.
- [x] Testes unitários cobrem: `done` (não `in_review`) sob shared branch, presença das instruções de PR consolidado, conteúdo legado preservado quando `IsSharedBranch=false`.
- [x] Testes existentes do feature-pipeline continuam passando (regression).

## Blocked by

- `.scratch/feature-pipeline/issues/05-branch-gate-and-task-payload.md`
- `.scratch/feature-pipeline/issues/06-daemon-branch-wiring-and-brief.md`

## Comments

### Key decisions made

1. **Gate keyed em `done`, não `in_review`** — A alternativa seria mudar o SQL do dependency gate pra aceitar `b.status IN ('done', 'in_review')`, mas isso quebraria casos legítimos onde o humano precisa revisar antes de liberar deps (PRD User Story #39). A solução é semântica: sob shared branch, a revisão NÃO é per-issue, então `in_review` deixa de fazer sentido naquele contexto. O agent pula esse estado.

2. **Comments no PR para mapear commits→issues** — Em vez de tentar editar o description do PR a cada push (race condition entre agents), cada agent posta um comentário curto no PR identificando seu commit. O reviewer escaneia os comentários do PR pra entender a estrutura. Description só lista todas as issues uma vez (no primeiro push), e edits subsequentes só acontecem se uma issue ficou de fora.

3. **Title feature-level, não issue-level** — Anteriormente o PR ficava com `feat(TES-1): ...`, confundindo o reviewer que via apenas a primeira issue mesmo com commits de TES-1, TES-2, TES-3 acumulados. Agora o título reflete a feature inteira; cada issue aparece nos comentários e no `## Implements`.

4. **Draft PR enquanto faltam issues** — O brief instrui o agent a marcar o PR como draft no primeiro push e tirar de draft no último push. Isso sinaliza ao humano "ainda não está pronto pra review", evitando reviews prematuras.

5. **`in_review` ainda vivo pra blockers reais** — Sob shared branch, `in_review` é reservado pra handoffs onde o agent não consegue terminar sem decisão humana (ex.: ambiguidade no escopo). Comment explicando o bloqueio é mandatory nesse caso.

### Files changed

- `server/internal/daemon/execenv/runtime_config.go` — Seção `## Shared branch` expandida com `### Consolidated PR model` e `### Status workflow under shared branch`. Passo 8 do assignment workflow agora é condicional.
- `server/internal/daemon/execenv/runtime_config_test.go` — Asserts atualizados em `TestSharedBranchSectionContent`; novos testes `TestSharedBranchProtocolMarksDoneNotInReview` e `TestSharedBranchPRConsolidationGuidance`.

### Blockers or notes for next iteration

None. Validado E2E pelo teste de feature "Conectar Telegram" no repo `WhaleBuddy/whalebuddy`:
- Antes do fix: A entregou PR #11 com title `feat(TES-1): add /integrations/telegram route`, ficou em `in_review`, B/C travadas até intervenção manual.
- Depois do fix: as próximas issues sob `IsSharedBranch=true` marcam `done` direto e o PR mantém title/description feature-level, pra revisão única no fim.

Próximo passo natural seria adicionar instruções similares pra issues isoladas (`IsSharedBranch=false`) que querem opt-in pro mesmo fluxo via `issue.metadata.target_branch`, mas isso já era suportado pelo Issue 02 e o brief atual ainda pula essas pelo `IsSharedBranch=false`. Considerar issue 14 se a feature for usada bastante com per-issue overrides.
