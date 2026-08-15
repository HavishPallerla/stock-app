import { Router } from 'express';
import { supabaseAdmin } from '../lib/supabaseAdmin.js';
import { requireAuth } from '../middleware/auth.js';

const router = Router();
router.use(requireAuth);

const VALID_SIGNALS = new Set(['buy', 'sell', 'hold']);

router.get('/', async (req, res) => {
  const { data, error } = await supabaseAdmin
    .from('alert_preferences')
    .select('sms_enabled, signal_types, updated_at')
    .eq('user_id', req.user.id)
    .maybeSingle();

  if (error) return res.status(500).json({ error: error.message });

  res.json({
    preferences: data ?? { sms_enabled: false, signal_types: ['buy', 'sell', 'hold'] },
  });
});

router.put('/', async (req, res) => {
  const { sms_enabled, signal_types } = req.body ?? {};

  if (typeof sms_enabled !== 'boolean') {
    return res.status(400).json({ error: 'sms_enabled must be a boolean' });
  }
  if (
    !Array.isArray(signal_types) ||
    signal_types.length === 0 ||
    !signal_types.every((s) => VALID_SIGNALS.has(s))
  ) {
    return res.status(400).json({ error: 'signal_types must be a non-empty array of buy/sell/hold' });
  }

  const { data, error } = await supabaseAdmin
    .from('alert_preferences')
    .upsert(
      { user_id: req.user.id, sms_enabled, signal_types, updated_at: new Date().toISOString() },
      { onConflict: 'user_id' },
    )
    .select('sms_enabled, signal_types, updated_at')
    .single();

  if (error) return res.status(500).json({ error: error.message });
  res.json({ preferences: data });
});

export default router;
