-- All writes to these tables happen server-side (Go scraper/analyzer, Express API)
-- using the service role key, which bypasses RLS entirely. The policies below only
-- govern what the frontend can do when it talks to Supabase directly (auth session +
-- optional Realtime subscription on tweet_analysis for the live dashboard feed).

alter table public.users enable row level security;
alter table public.watchlist enable row level security;
alter table public.alert_preferences enable row level security;
alter table public.tweets enable row level security;
alter table public.tweet_analysis enable row level security;
alter table public.sms_log enable row level security;

create policy "users read own row" on public.users
  for select using (auth.uid() = id);

create policy "users read own watchlist" on public.watchlist
  for select using (auth.uid() = user_id);

create policy "users read own alert preferences" on public.alert_preferences
  for select using (auth.uid() = user_id);

create policy "users read own sms log" on public.sms_log
  for select using (auth.uid() = user_id);

-- Tweets and analysis are non-sensitive, aggregate market signal data: any signed-in
-- user can read the live feed (e.g. via Supabase Realtime) directly from the client.
create policy "authenticated users read tweets" on public.tweets
  for select using (auth.role() = 'authenticated');

create policy "authenticated users read tweet analysis" on public.tweet_analysis
  for select using (auth.role() = 'authenticated');
