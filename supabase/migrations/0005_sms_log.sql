create table if not exists public.sms_log (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users (id) on delete cascade,
  tweet_analysis_id uuid not null references public.tweet_analysis (id) on delete cascade,
  status text not null check (status in ('sent', 'failed', 'skipped')),
  error_message text,
  sent_at timestamptz not null default now()
);

create index if not exists sms_log_user_id_idx on public.sms_log (user_id);
create index if not exists sms_log_tweet_analysis_id_idx on public.sms_log (tweet_analysis_id);
