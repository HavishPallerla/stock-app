import { supabaseAdmin } from '../lib/supabaseAdmin.js';
import { asyncHandler } from '../lib/asyncHandler.js';

// Verifies the Supabase access token issued by the frontend's Supabase Auth
// session and attaches the resolved user to the request. The frontend never
// talks to Postgres directly for anything auth-gated below — it sends this
// token to Express, and Express does the DB work with the service role key.
//
// Wrapped in asyncHandler because Express 4 does not await middleware or
// forward a rejected promise to the error handler on its own — an unwrapped
// async middleware that throws would hang the request instead of failing.
export const requireAuth = asyncHandler(async (req, res, next) => {
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
});
