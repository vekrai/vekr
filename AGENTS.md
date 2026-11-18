# Vekr

Personal learning agent: knowledge management + adaptive learning, client/server, solo-first with multi-user planned later. The assistant persona is "V".

## Working with the user

- The user is a Rails developer, new to Go. Build one feature at a time, small enough to pause and explain at each checkpoint: implement → verify → pause for questions.
- Never run `git add` / `git commit` (not even to inspect a staged diff).
- Don't add code comments.

## Tech stack

- **Go** via `mise` (pinned in `mise.toml`), plain `net/http` `ServeMux` (Go 1.22+ method + `{param}` patterns), hand-written `internal/middleware.go` (logging, session, panic recovery). No chi/gin.
- **Templates**: `templ` (`.templ` → generated `_templ.go`, both committed). Regenerate after every `.templ` edit.
- **DB**: Postgres, `pgx` + GORM. Migrations via `golang-migrate`, plain `.sql` in `db/migrations/`.
- **Live updates**: SSE (`<turbo-stream-source>`) + in-process pub/sub in `internal/live`. Single instance, no Redis, no websockets.
- **Background work**: chat replies run in a goroutine (in-flight replies are lost on restart). River (Postgres-backed queue) is planned for curriculum/practice generation; `internal/jobs` is still empty.
- **Search**: Postgres full-text search via GIN expression index on `to_tsvector('english', title || ' ' || content)` (plus a trigram index). No stored tsvector column, no embeddings, no `pgvector`. CJK tokenization is a known, accepted limitation.
- **Sessions**: `alexedwards/scs`.
- **i18n**: `nicksnyder/go-i18n`, YAML in `locales/`, `t(ctx, key)` in templates. Locale from the single-row `settings` table (`en`/`zh`, toggled by the A/文 button via `POST /locale`). All UI strings go through `locales/en.yaml`; `locales/zh.yaml` mirrors it. Every agent stage adds `prompts/language_system.txt` as a system message so V writes in the chosen language.
- **Frontend**: Hotwire (Turbo + Stimulus) via import maps, vendored into `web/static/js/`, no bundler. Versions pinned in `package.json` (metadata only); `mise run js:vendor` re-fetches after a version bump. Stimulus controllers in `web/static/js/controllers/`.
- **CSS**: Tailwind v4 standalone CLI. Design tokens via `@theme` in `web/static/css/input.css`; `web/static/css/app.css` is the generated, committed output.

## Commands

```
mise run dev                  # templates:generate + css:build, then go run ./cmd/server
mise run templates:generate   # templ generate
mise run css:build            # rebuild app.css
mise run db:migrate
mise run db:seed
mise run db:reset             # down -all, migrate, seed (respects APP_ENV)
mise run test                 # go test ./... with APP_ENV=test
```

## Layout

```
cmd/server, cmd/seed
internal/
  models/     one file per table, colocated _test.go
  handlers/   dashboard, notes, chat, practice, streams
  ai/         provider layer (OpenAI-compatible protocol + config)
  agent/      orchestration: chat, distill, curriculum, practice, grading, tools, structured, prompts/
  live/       pub/sub hub for Turbo Stream broadcasts
  i18n/, config/, markdown/, jobs/
  router.go, middleware.go, app.go
web/templates/, web/static/{js,css}
db/migrations/
locales/en.yaml
```

## Database conventions

- **No foreign keys.** Never use `REFERENCES` / FK constraints in migrations. Plain `BIGINT` id columns; referential cleanup happens in Go. (Existing FKs were dropped in `drop_all_foreign_keys`.)
- Nullable columns are Go pointer types.
- `user_id` is planned on every row and threaded through every call/write, even while solo.

## Core design: Topics group notes

A **Topic** (`topics` table) is the unit of learning: it owns the status/mastery, the practice items, and the concepts. **Notes** (`nodes`) are plain content; a note optionally belongs to one topic via `nodes.topic_id`. "Turn into topic" on a note creates a Topic from it and assigns the note to it. Notes share one graph (`node_links`) and one search index regardless of topic.

## Feature pipeline

```
Intake → Curriculum Builder → Scheduler ⇄ (Learning Session + Practice & Assessment)
```

1. **Intake** — chat with V, or paste/upload/link material.
2. **Curriculum Builder** — turns intake into one Topic plus outline notes with prerequisite links between them; user confirms before committing.
3. **Scheduler** — topics + mastery state → the dashboard Queue.
4. **Learning Session** — per-topic read/explore (`/topics/{id}`) + explain-it-back; the result is saved as a Note.
5. **Practice & Assessment** — generated questions, agent-graded (qualitative review, not just pass/fail), updates mastery; weak results feed back into the Queue.

## Data model

Migrations in `db/migrations/` are the source of truth. Domain tables:

- `topics` — `title`, `content`, `status` (`not_started`/`learning`/`practicing`/`mastered`). Status is set by grading: `practicing` after any attempt, `mastered` once every item's latest attempt passed with no mistake logged (terminal).
- `nodes` — `title`, `content`, `topic_id` (optional), `source_chat_id`.
- `node_links` — knowledge graph + prerequisite edges between notes, distinguished by `relation` (`related`, `prerequisite_of`, `source_of`, …).
- `chats` (optional `node_id`), `chat_messages` — `role` (`user`/`agent`/`compaction`, plus tool fields).
- `concepts` — fine-grained skill tags (e.g. "boundary-pointers") scoped by `topic_id`. They drive Weak Spots, mistake attribution, and interleaved drills across topics.
- `practice_items` — `topic_id`, `type` (`mcq`/`flashcard`/`open`/`code`), `prompt`, `answer_key`; `practice_item_concepts` (m:n). Generated from the topic's content plus all its notes.
- `attempts` — `submission`, `passed`, `agent_feedback`. `passed` and feedback are independent: an attempt can pass and still log a mistake (e.g. fragile boundary logic that happens to look right).
- `mistake_log` — `attempt_id`, `concept_id`, `error_type` (`misconception`/`careless_slip`/`knowledge_gap`/`calculation_error`).
- `concept_weakness` — materialized rollup (`miss_count`, `attempt_count`, `last_missed_at`) per `(user_id, concept_id)`, recomputed on `mistake_log` insert.

Planned, not yet migrated:

- `users`.
- `ai_configs` — one row per role (`chat`, or a stage override): `base_url`, `api_key`, `model`, `context_window`.
- `ai_calls` — one row per request/response: `turn_id` + `attempt_number` + `outcome` (`success`/`retry`/`gave_up`), `stage` (`chat`/`distill`/`curriculum`/`practice`/`grading`/`compact`), prompt, response, `finish_reason`, tokens, `parse_success`, `latency_ms`. A retried call is two rows sharing a `turn_id`.
- `tool_calls` — `chat_message_id`, `tool_name`, `args` (jsonb), `result`, `status`.
- `job_claims` — binary crash-recovery marker for River jobs (`released_at` null on boot = crashed, redo from scratch).

Note keywords (tag chips) come from `ts_stat()` over the same tsvector expression — no LLM, no stored column.

## Agent & AI design

**Provider layer (`internal/ai`)** — one client for any OpenAI-compatible `/v1/chat/completions` endpoint (local llama.cpp serving Qwen, DeepSeek, …). Protocol (request shaping, SSE parsing, tool-call/finish-reason extraction) is written once; providers are config data, not code.

**Orchestration (`internal/agent`)** — five single-purpose stages: chat, distill, curriculum, practice, grading. Distill/curriculum/practice/grading are one-shot structured-output calls sequenced by Go — no autonomous loop.

**Chat is the one tool-calling loop.** Call model → dispatch tool calls → append results → repeat, capped at 8 steps per message. Every tool call is schema-validated before dispatch; failures are fed back as correction turns, capped. History is normalized so every tool call has a result (dangling calls get a synthetic "no result").

**Structured-output helper (`structured.go`)** — shared retry-with-correction: feed parse errors back, cap attempts, distinguish truncation (`length`) from malformed output, give up loudly.

**Tools** — no per-call permission gate; logging is the control.
- `websearch` — Tavily only. `api_key` optional: keyless requests use Vekr's own client name (never impersonate another product); if rejected, fail clearly asking for a key.
- `webfetch` — http/https only, size cap enforced while streaming, timeout, redirects followed one hop at a time with the safety check re-run per hop, textual content-type allowlist.
- `search_notes(query)` — `plainto_tsquery` + `ts_rank`, returns `{node_id, title, snippet}` (`ts_headline`).
- `read_note(node_id)` — full content of one note.
- `generate_curriculum_preview` / `commit_curriculum_preview` — Topic + outline notes, preview held in memory per chat until confirmed.
- `generate_practice_items(topic_id)` — commits directly. The model finds `topic_id` from `search_notes` results or the note page's chat context.

**Context / compaction** — stateless: resume = replay `chat_messages` for a chat. When the latest call's tokens / `context_window` crosses 95%, summarize into a `role = 'compaction'` message; later calls replay the latest compaction row + everything after it. No turn-count cap. The chat panel shows live usage, e.g. "12.4k / 32k tokens · 39%".

**Guardrails**
- Preview-then-commit only for graph-shaped output (curriculum: Topic + notes + links), held in memory/session, never a persisted pending state. Single-row outputs (note, practice item, grade) commit directly.
- Log every provider call and tool call.
- Grading never executes code — `code` items are judged by the LLM, same as `open`. `mcq`/`flashcard` are exact-match.

## Scheduling policy

- No auto-computed priority. The Queue is non-mastered topics ordered by latest activity (topic update or attempt); the user picks what to do next.
- `concept_weakness` is informational only (Weak Spots card), not a reordering signal.
- No forgetting-curve resurfacing; "mastered" is terminal for now.
- Drills are interleaved: pull practice items sharing the weak concept (`practice_item_concepts` by `concept_id`), round-robin across topics.
- "Start today's session" opens the first Queue item in its matching view; no multi-step session runner.

## Dashboard

- Top bar: logo + theme toggle (no separate search/ask — the chat box owns that).
- Today's Queue, Chat with V (single intake point; opens a chat modal), Recent Notes, Topics (passed / total practice items), Weak Spots (with "generate drill"), Mastered topics, Knowledge Graph preview, Stats.

## UI direction

Calm, neutral, low-distraction:
- Neutral surfaces, thin hairline borders instead of heavy shadows.
- One desaturated accent, only for progress fills, links, and weak-topic markers.
- Primary buttons and logo are monochrome; color is reserved for status/progress, never chrome.
- Status colors (good/warning/serious/critical) are a separate palette from the accent.
- Typography and spacing carry hierarchy.
- Dark mode is a deliberate second pass, not an invert.

## Explicitly not building (yet)

- Multi-user accounts/sharing/permissions (single user for now).
- A full calendar app or external calendar sync — only a lightweight in-app day/week view.
- Auto priority scoring, spaced-repetition resurfacing, mastery decay.
- Per-domain generation templates — one generic generator.
- Autonomous tool loops outside chat; step-by-step execution checkpointing; a code-execution sandbox.
