-- Profile table mirroring auth.users. We keep our own `users` table (rather than
-- relying on auth.users directly) so the rest of the schema, and the Express API,
-- never needs to touch the `auth` schema.
create table if not exists public.users (
  id uuid primary key references auth.users (id) on delete cascade,
  email text unique not null,
  phone text,
  created_at timestamptz not null default now()
);

-- Keep public.users in sync whenever someone signs up via Supabase Auth.
create or replace function public.handle_new_auth_user()
returns trigger
language plpgsql
security definer set search_path = public
as $$
begin
  insert into public.users (id, email, phone)
  values (new.id, new.email, new.phone)
  on conflict (id) do nothing;
  return new;
end;
$$;

drop trigger if exists on_auth_user_created on auth.users;
create trigger on_auth_user_created
  after insert on auth.users
  for each row execute function public.handle_new_auth_user();
