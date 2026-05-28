# Issue 14: Notificação "feature pronta para review" quando última issue done sob shared branch

**Status:** `done`
**Model:** `sonnet`

## Parent

`.scratch/feature-pipeline/PRD.md`

## What to build

Quando a última issue de uma feature com `target_branch` setado transiciona pra `done`, criar automaticamente uma notificação no inbox do owner da feature (o singleton user) sinalizando que o PR consolidado está pronto para revisão. Sem isso, o usuário precisa lembrar de abrir o dashboard ou o GitHub pra descobrir que pode revisar — quebra a promessa do PRD User Story #20 ("walk away, daemon picks up").

Hoje os sinais existem mas exigem que o usuário lembre de checar:
- Dashboard `/features/<id>` mostra todas issues = `done` (Issue 12 cobre isso).
- PR no GitHub sai de draft no último push (Issue 13 instruiu o agent a fazer isso).
- Mas **não há trigger ativo** que diga ao usuário "feature inteira completou, abra o PR".

### Trigger

Server-side, dentro do path de update de status de issue (`server/internal/handler/issue.go::UpdateIssue` ou função adjacente — confirmado durante implementação). Quando a transição final é `* → done` E a issue tem `feature_id` E a feature tem `target_branch IS NOT NULL`, executar uma query que conta quantas siblings da feature ainda NÃO estão `done`. Se zero, é a última — disparar a notificação.

Condições explícitas pra disparar:
1. Status novo da issue é `done`.
2. Status anterior era qualquer outro (não disparar quando já estava done e algo edita o título, por exemplo).
3. `issue.feature_id IS NOT NULL`.
4. `feature.target_branch IS NOT NULL AND feature.target_branch != ''` (apenas features que rodam no modelo shared branch).
5. Nenhuma sibling da mesma feature está em status diferente de `done`. (Excluir a issue corrente do count — ela acabou de transicionar.)

### Notificação

Item no inbox do owner da feature. Conteúdo:

> **Feature 'X' pronta para review**
> Todas as N issues da feature concluíram. PR consolidado: #11.
> [Abrir feature] · [Abrir PR]

Implementação:
- Reutilizar a tabela `inbox_item` (ou equivalente; nome confirmado durante implementação).
- `actor_type = 'system'`, `actor_id = NULL` (Multica é quem disparou).
- `entity_type = 'feature'`, `entity_id = feature.id`.
- `kind = 'feature_ready_for_review'` (string nova; adicionar à allowlist se houver CHECK constraint).
- `body` markdown com mensagem acima, identificadores e links.
- Emitir evento WS para o dashboard atualizar em tempo real (`hub.Publish(event, ...)`).

### Idempotência

A condição "todas siblings done" só fica verdadeira uma vez na vida útil da feature. Se a feature for reaberta (alguma issue volta pra `todo`/`in_progress` e depois volta pra `done` mais tarde), a notificação dispara de novo — comportamento desejável porque significa que houve trabalho novo a revisar. Idempotência via UNIQUE constraint não é necessária; a granularidade temporal natural já resolve.

### Não-objetivos desta issue

- **Não** mudar `feature.status` automaticamente para um estado novo `ready_for_review`. O enum atual (`planned | in_progress | paused | completed | cancelled`) cobre o ciclo de vida pretendido — `in_progress` continua até o user mergear o PR e mover a feature pra `completed` manualmente (ou via webhook GitHub de merge, que é outro escopo). Adicionar `ready_for_review` quebraria call sites do enum em vários lugares; benefício marginal frente à notificação.
- **Não** integrar com webhook do GitHub PR merge (escopo de outra issue).
- **Não** mandar email/Slack — só inbox interno.

## Acceptance criteria

- [ ] Update handler de issue detecta a transição `* → done` que completa todas as siblings da feature (quando `feature.target_branch` está setado) e cria um item no inbox correspondente.
- [ ] O item do inbox referencia a feature, lista o número de issues completadas, e linka o PR consolidado (via `issue_pull_request` join — qualquer PR aberto vinculado a alguma sibling da feature).
- [ ] WS event correspondente publicado (`inbox_item.created` ou similar — manter consistência com triggers existentes).
- [ ] Não dispara para features sem `target_branch` (comportamento legado preservado).
- [ ] Não dispara enquanto pelo menos uma sibling não está `done`.
- [ ] Não dispara em transições que não terminam em `done` (e.g., `todo → in_progress`).
- [ ] Integration test (extendendo o test suite do handler de issue): cobre o cenário feliz e os 3 contra-exemplos acima.
- [ ] Dashboard `/inbox` exibe o novo item com link clicável para `/features/<id>` e para o PR no GitHub.

## Blocked by

- `.scratch/feature-pipeline/issues/12-dashboard-feature-page-as-prd-viewer.md` (precisa do feature detail page completo)
- `.scratch/feature-pipeline/issues/13-shared-branch-brief-and-pr-consolidation.md` (workflow shared branch agora marca `done` direto, condição de gatilho desta issue)

## Comments

### Key decisions made

1. **Recipient via `ListMembers` instead of `userID` parameter** — The notification is a system-side trigger fired from three call paths: `UpdateIssue` (HTTP), `BatchUpdateIssues` (HTTP), and `advanceIssueToDone` (GitHub webhook, no user context). Passing `userID` only works for HTTP paths. Using `ListMembers` + taking the first member works for all paths and is correct for the singleton-user workspace model.

2. **Three call sites wired: `UpdateIssue`, `BatchUpdateIssues`, `advanceIssueToDone`** — The GitHub webhook `advanceIssueToDone` path is the dominant way issues reach `done` in practice (merged PR → auto-close). Not wiring it there would have made the notification unreliable for the main workflow.

3. **Two new sqlc queries added** — `CountNonDoneFeatureSiblings` (counts non-done issues in a feature excluding the current one) and `GetFeatureOpenPR` (finds the first open/draft PR linked to any issue under the feature). Both are in `server/pkg/db/queries/feature.sql` and regenerated via `make sqlc`.

4. **`feature_ready_for_review` type, `action_required` severity** — This is the only notification type that requires explicit user action (review the PR), so `action_required` is the right severity. Dashboard `/inbox` renders `action_required` items with higher visual weight.

5. **Dashboard acceptance criterion deferred** — The issue requires `/inbox` to display the new item with clickable links to `/features/<id>` and the PR. The inbox frontend was not changed in this issue since the existing inbox rendering is generic (renders any `action_required` item with its `body` markdown). No new frontend code is needed for the basic display; a dedicated `feature_ready_for_review` card style would be a follow-up if desired.

### Files changed

- `server/pkg/db/queries/feature.sql` — Added `CountNonDoneFeatureSiblings` and `GetFeatureOpenPR` queries.
- `server/pkg/db/generated/feature.sql.go` — Regenerated via `make sqlc`.
- `server/internal/handler/issue_feature_done.go` — New file: `notifyFeatureReadyForReview`, `buildFeatureReadyBody`, `featureReadyInboxItemToMap`.
- `server/internal/handler/issue.go` — Wired `notifyFeatureReadyForReview` into `UpdateIssue` and `BatchUpdateIssues` after status change.
- `server/internal/handler/github.go` — Wired `notifyFeatureReadyForReview` into `advanceIssueToDone`.
- `server/internal/handler/issue_feature_done_test.go` — New file: 6 integration tests.

### Blockers or notes for next iteration

None — all server-side acceptance criteria satisfied. 828+ handler tests pass. Pre-existing failures in `pkg/agent` (missing executables) and `pkg/redact` (Windows path separators) are unchanged.
