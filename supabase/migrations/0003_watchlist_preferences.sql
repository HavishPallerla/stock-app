create table if not exists public.watchlist (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users (id) on delete cascade,
  ticker text not null,
  created_at timestamptz not null default now(),
  unique (user_id, ticker)
);

create index if not exists watchlist_user_id_idx on public.watchlist (user_id);
create index if not exists watchlist_ticker_idx on public.watchlist (ticker);

create table if not exists public.alert_preferences (
  user_id uuid primary key references public.users (id) on delete cascade,
  sms_enabled boolean not null default false,
  signal_types text[] not null default array['buy', 'sell', 'hold'],
  updated_at timestamptz not null default now()
);
