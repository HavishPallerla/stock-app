import { createClient } from '@supabase/supabase-js';

const url = import.meta.env.VITE_SUPABASE_URL;
const anonKey = import.meta.env.VITE_SUPABASE_ANON_KEY;

if (!url || !anonKey) {
  // eslint-disable-next-line no-console
  console.warn('VITE_SUPABASE_URL / VITE_SUPABASE_ANON_KEY are not set — see .env.example');
}

// Anon key only. This client is used for auth (sign up/in/out, session
// handling) and, once signed in, for the Realtime subscription on the
// dashboard's live feed. All other data access goes through the Express API,
// which holds the service role key.
export const supabase = createClient(url, anonKey);
