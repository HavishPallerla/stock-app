import { supabaseAdmin } from '../lib/supabaseAdmin.js';

// Verifies the Supabase access token issued by the frontend's Supabase Auth
// session and attaches the resolved user to the request. The frontend never
// talks to Postgres directly for anything auth-gated below — it sends this
// token to Express, and Express does the DB work with the service role key.
export async function requireAuth(req, res, next) {
  const header = req.headers.authorization ?? '';
  const token = header.startsWith('Bearer ') ? header.slice('Bearer '.length) : null;

  if (!token) {
    return res.status(401).json({ error: 'Missing bearer token' });
  }

  const { data, error } = await supabaseAdmin.auth.getUser(token);
  if (error || !data?.user) {
    return res.status(401).json({ error: 'Invalid or expired session' });
  }

  req.user = { id: data.user.id, email: data.user.email };
  next();
}
