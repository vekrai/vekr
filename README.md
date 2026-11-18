# Vekr

A personal learning agent. Chat with **V** to collect knowledge into notes, turn notes into topics, practice them with generated questions, and let the agent grade your answers and track your weak spots.

```
Intake → Curriculum Builder → Scheduler ⇄ (Learning Session + Practice & Assessment)
```

- **Notes** are plain markdown content, linked into one knowledge graph and searchable with Postgres full-text search.
- **Topics** group notes and are the unit of learning: each has a status (`not started` → `learning` → `practicing` → `mastered`) and its own practice items.
- **Practice** items (multiple choice, flashcard, open, code) are generated from a topic and graded by the LLM, with mistakes attributed to fine-grained concepts.
- **Dashboard** shows today's queue, topics and progress, recent notes and chats, weak spots with targeted drills, and stats.
- **Chat with V** runs a tool-calling agent that can search and read your notes, search and fetch the web, build a curriculum, and generate practice items.
- **English / 中文** UI, switched with the A/文 button in the top bar.

## Stack

Go (`net/http`), `templ` templates, Postgres (`pgx` + GORM, `golang-migrate`), Hotwire (Turbo + Stimulus via import maps), Tailwind v4, SSE for live updates. The LLM is any OpenAI-compatible `/v1/chat/completions` endpoint, such as llama.cpp or Ollama running locally, or a hosted provider.

## Requirements

- [mise](https://mise.jdx.dev/) installs Go, templ, Tailwind, and golang-migrate at the versions pinned in `mise.toml`
- PostgreSQL
- An OpenAI-compatible LLM endpoint

## Setup

```sh
mise install
cp .envrc.example .envrc
```

Edit `.envrc`, then load it (with [direnv](https://direnv.net/), or `source .envrc`):

| Variable | Default | Purpose |
| --- | --- | --- |
| `DATABASE_USERNAME` / `DATABASE_PASSWORD` / `DATABASE_HOST` | | Postgres connection |
| `APP_ENV` | `development` | Selects the database `vekr_${APP_ENV}` |
| `PORT` | `8000` | HTTP port |
| `AI_BASE_URL` | `http://localhost:11434/v1` | OpenAI-compatible endpoint |
| `AI_MODEL` | `qwen` | Model name |
| `AI_API_KEY` | | API key, if the endpoint needs one |
| `AI_MAX_TOKENS_FIELD` | `max_tokens` | Override for providers that use e.g. `max_completion_tokens` |
| `TAVILY_API_KEY` | | Optional key for the `websearch` tool |

Create the databases and run migrations:

```sh
createdb vekr_development
createdb vekr_test
mise run db:migrate
APP_ENV=test mise run db:migrate
```

## Running

```sh
mise run dev
```

Open http://localhost:8000. This watches `.templ`, Go, CSS, and locale files and reloads the browser on change. Use `mise run serve` to run once without watching.

## Using Vekr

1. **Chat with V** from the box in the top bar. Ask questions, paste material, or ask V to build a topic on something you want to learn.
2. **Confirm the curriculum** V previews: a topic plus outline notes with prerequisite links.
3. **Read** the topic page (`/topics/{id}`) and its notes. Any note can also be turned into a topic.
4. **Practice**: ask V to generate practice items for a topic, then answer them. V grades each attempt with feedback and logs mistakes per concept.
5. **Review** the dashboard: the queue orders unfinished topics by latest activity, and Weak Spots offers drills that mix items sharing a weak concept across topics.
6. **Save a chat as a note** to keep what you learned in the knowledge graph.

## Commands

```sh
mise run dev                  # generate templates + CSS, run with live reload
mise run serve                # run once, no watching
mise run templates:generate   # templ generate
mise run css:build            # rebuild web/static/css/app.css
mise run db:migrate           # apply migrations
mise run db:seed              # seed data
mise run db:reset             # roll back all migrations, migrate, seed
mise run test                 # go test ./... against vekr_test
mise run js:vendor            # re-fetch vendored Turbo/Stimulus after a version bump
```

`db:migrate` and `db:reset` act on the database selected by `APP_ENV`, e.g. `APP_ENV=test mise run db:reset`.

## Project layout

```
cmd/server, cmd/seed     entry points
internal/
  handlers/              HTTP handlers
  models/                one file per table
  agent/                 chat loop, distill, curriculum, practice, grading, tools, prompts
  ai/                    OpenAI-compatible client
  live/                  pub/sub for Turbo Stream broadcasts
  i18n/, markdown/, config/
  router.go, middleware.go, app.go
web/templates/           .templ files and generated _templ.go
web/static/              vendored JS, Stimulus controllers, CSS
db/migrations/           plain SQL migrations
locales/                 en.yaml, zh.yaml
```

Generated files (`*_templ.go`, `web/static/css/app.css`) are committed. Regenerate them after editing `.templ` files or CSS.

## Adding a language

All UI strings go through `t(ctx, key)` in templates, and a test checks that every key used in a template exists in the catalog. The current language is stored in the single-row `settings` table and flipped between `en` and `zh` by `POST /locale`.

To add another language: add `locales/<lang>.yaml` with the same keys as `locales/en.yaml`, extend the `locale` CHECK constraint on `settings` with a new migration, and replace the two-way toggle with a picker.
