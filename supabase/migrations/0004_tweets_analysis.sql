-- "tweets" holds scraped financial news items. The scraper's current source is
-- RSS feeds (see backend-go/internal/feed), not literal tweets, but we keep the
-- table/column names from the original spec: source_account is the feed/publisher
-- name (e.g. "Reuters Business"), content is the headline + summary text.
create table if not exists public.tweets (
  id uuid primary key default gen_random_uuid(),
  external_id text not null unique, -- RSS guid/link, used for de-duplication
  source_account text not null,
  title text,
  content text not null,
  url text,
  published_at timestamptz,
  scraped_at timestamptz not null default now(),
  processed boolean not null default false
);

create index if not exists tweets_processed_idx on public.tweets (processed) where processed = false;
create index if not exists tweets_scraped_at_idx on public.tweets (scraped_at desc);

create table if not exists public.tweet_analysis (
  id uuid primary key default gen_random_uuid(),
  tweet_id uuid not null references public.tweets (id) on delete cascade,
  ticker text not null,
  signal text not null check (signal in ('buy', 'sell', 'hold')),
  confidence numeric(4, 3) not null check (confidence >= 0 and confidence <= 1),
  justification text,
  created_at timestamptz not null default now()
);

create index if not exists tweet_analysis_ticker_idx on public.tweet_analysis (ticker);
create index if not exists tweet_analysis_created_at_idx on public.tweet_analysis (created_at desc);
create index if not exists tweet_analysis_tweet_id_idx on public.tweet_analysis (tweet_id);
