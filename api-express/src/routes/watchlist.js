import { Router } from 'express';
import { supabaseAdmin } from '../lib/supabaseAdmin.js';
import { requireAuth } from '../middleware/auth.js';
import { asyncHandler } from '../lib/asyncHandler.js';

const router = Router();
router.use(requireAuth);

const TICKER_RE = /^[A-Z.]{1,10}$/;

router.get('/', asyncHandler(async (req, res) => {
  const { data, error } = await supabaseAdmin
    .from('watchlist')
    .select('id, ticker, created_at')
    .eq('user_id', req.user.id)
    .order('created_at', { ascending: false });

  if (error) return res.status(500).json({ error: error.message });
  res.json({ watchlist: data });
}));

router.post('/', asyncHandler(async (req, res) => {
  const ticker = String(req.body?.ticker ?? '').trim().toUpperCase();
  if (!TICKER_RE.test(ticker)) {
    return res.status(400).json({ error: 'ticker must be 1-10 uppercase letters (e.g. AAPL)' });
  }

  const { data, error } = await supabaseAdmin
    .from('watchlist')
    .upsert({ user_id: req.user.id, ticker }, { onConflict: 'user_id,ticker' })
    .select('id, ticker, created_at')
    .single();

  if (error) return res.status(500).json({ error: error.message });
  res.status(201).json({ watchlist: data });
}));

router.delete('/:ticker', asyncHandler(async (req, res) => {
  const ticker = String(req.params.ticker).toUpperCase();

  const { error } = await supabaseAdmin
    .from('watchlist')
    .delete()
    .eq('user_id', req.user.id)
    .eq('ticker', ticker);

  if (error) return res.status(500).json({ error: error.message });
  res.status(204).end();
}));

export default router;
