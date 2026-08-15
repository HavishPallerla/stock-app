import { Router } from 'express';
import { supabaseAdmin } from '../lib/supabaseAdmin.js';
import { requireAuth } from '../middleware/auth.js';

const router = Router();
router.use(requireAuth);

const VALID_SIGNALS = new Set(['buy', 'sell', 'hold']);
const MAX_LIMIT = 100;

// Recent signals for the dashboard's live feed / history view.
router.get('/', async (req, res) => {
  const limit = Math.min(Number(req.query.limit) || 50, MAX_LIMIT);

  let query = supabaseAdmin
    .from('tweet_analysis')
    .select(
      'id, ticker, signal, confidence, justification, created_at, tweets(title, content, url, source_account, published_at)',
    )
    .order('created_at', { ascending: false })
    .limit(limit);

  if (req.query.ticker) {
    query = query.eq('ticker', String(req.query.ticker).toUpperCase());
  }
  if (req.query.signal) {
    const signal = String(req.query.signal).toLowerCase();
    if (!VALID_SIGNALS.has(signal)) {
      return res.status(400).json({ error: 'signal must be buy, sell, or hold' });
    }
    query = query.eq('signal', signal);
  }

  const { data, error } = await query;
  if (error) return res.status(500).json({ error: error.message });
  res.json({ signals: data });
});

export default router;
