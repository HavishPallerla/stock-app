# stock-app

Real-time stock alerts: scrapes financial news RSS feeds, runs each item through Claude for a
buy/sell/hold trading signal, and delivers matches to users via SMS (Twilio) and a live web
dashboard.

```
supabase/      SQL migrations — schema for users, watchlist, tweets, analysis, sms_log
backend-go/    Go worker: RSS scraper (Colly) -> Claude analysis -> Supabase -> Twilio alerts
api-express/   Express BFF between the frontend and Supabase (holds the service role key)
frontend/      React (Vite) dashboard + onboarding + settings
```

## How data flows

1. **`backend-go`** polls RSS feeds (or a bundled mock fixture, by default) with Colly, dedupes
   against `tweets.external_id`, and pushes new items onto an in-memory channel.
2. A worker pool calls **Claude** (tool-use, forced structured output) for each item, gets back
   ticker/signal/confidence/justification, and writes rows to `tweet_analysis`.
3. For each signal, it looks up users whose `watchlist` + `alert_preferences` match and sends SMS
   via **Twilio** (behind the `FEATURE_SMS_ALERTS` flag), logging to `sms_log`.
4. **`api-express`** exposes REST endpoints for watchlist/preferences/profile CRUD and recent
   signals, authenticated via Supabase Auth JWTs, using the service role key server-side only.
5. **`frontend`** handles auth + onboarding via the Supabase client SDK directly, then talks to
   `api-express` for everything else. The dashboard also subscribes to Supabase Realtime on
   `tweet_analysis` inserts for a live feed.

## Where the latency budget goes (target: ~2.5s/tweet end to end)

Every processed tweet logs a stage breakdown (`backend-go/internal/pipeline`):
`queue_wait_ms` (scrape → picked up by a worker), `claude_ms` (the LLM call — normally the
dominant cost), `db_write_ms` (analysis + processed-flag writes), `alerts_ms` (SMS fan-out).
Watch these in the worker's JSON logs before tuning anything — Claude latency and DB round-trips
are the two stages actually worth optimizing; the scrape step is decoupled via the channel and
isn't on this critical path per item.

## Local setup

Each service has its own `.env.example` — copy to `.env` and fill in real values. Nothing here
runs against a real Supabase project until you create one and run the migrations in
`supabase/migrations/` against it (e.g. via the Supabase SQL editor, or `supabase db push` if
using the Supabase CLI).

```bash
# 1. Supabase: apply supabase/migrations/*.sql to your project, grab the project URL +
#    anon key + service role key.

# 2. Go worker (defaults to the bundled mock feed, SMS alerts off)
cd backend-go && cp .env.example .env   # fill in SUPABASE_URL / SUPABASE_SERVICE_ROLE_KEY / ANTHROPIC_API_KEY
go run ./cmd/worker

# 3. Express API
cd api-express && cp .env.example .env  # fill in SUPABASE_URL / SUPABASE_SERVICE_ROLE_KEY
npm install && npm run dev

# 4. Frontend
cd frontend && cp .env.example .env     # fill in VITE_SUPABASE_URL / VITE_SUPABASE_ANON_KEY
npm install && npm run dev
```

Toggle `backend-go`'s `FEATURE_LIVE_FEED=true` (with `FEED_URLS` set) once you're ready to move
off the mock fixture, and `FEATURE_SMS_ALERTS=true` (with Twilio credentials) once you're ready
to spend real SMS money.

## Notes / open decisions

- **Tweet source**: X/Twitter scraping is off the table (ToS + auth cost). The scraper targets
  RSS feeds from financial publishers instead (Reuters, MarketWatch, Business Wire, Benzinga,
  Yahoo Finance, ...) — same "financial news firehose" shape, no scraping legality risk. Swap in
  more feeds by adding to `FEED_URLS`; swapping the source entirely means implementing a new
  `feed.Source` in `backend-go/internal/feed`.
- **Phone verification**: `PUT /api/profile/phone` currently sets the number with no OTP check —
  Twilio Verify integration was intentionally deferred with the rest of the SMS work (build order
  step 6) so the app is fully testable before any SMS spend.
- **Model choice**: the worker defaults to `claude-haiku-4-5-20251001` for the per-tweet
  classification call — it's the cheap/fast option and this is a latency-sensitive, low-ambiguity
  structured extraction task. Swap `ANTHROPIC_MODEL` to a Sonnet model if you want higher-quality
  justifications at the cost of latency.
