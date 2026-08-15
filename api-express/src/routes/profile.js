import { Router } from 'express';
import { supabaseAdmin } from '../lib/supabaseAdmin.js';
import { requireAuth } from '../middleware/auth.js';

const router = Router();
router.use(requireAuth);

// E.164, e.g. +15555550123 — the format Twilio expects.
const PHONE_RE = /^\+[1-9]\d{7,14}$/;

router.get('/', async (req, res) => {
  const { data, error } = await supabaseAdmin
    .from('users')
    .select('id, email, phone, created_at')
    .eq('id', req.user.id)
    .single();

  if (error) return res.status(500).json({ error: error.message });
  res.json({ profile: data });
});

// NOTE: this sets the phone number without OTP verification. Real phone
// verification (Twilio Verify) ships with COMPONENT 6 / the SMS feature
// flag — see backend-go/.env.example FEATURE_SMS_ALERTS. Until then this is
// a plain field update so onboarding + the rest of the app can be built and
// tested end to end.
router.put('/phone', async (req, res) => {
  const phone = String(req.body?.phone ?? '').trim();
  if (!PHONE_RE.test(phone)) {
    return res.status(400).json({ error: 'phone must be in E.164 format, e.g. +15555550123' });
  }

  const { data, error } = await supabaseAdmin
    .from('users')
    .update({ phone })
    .eq('id', req.user.id)
    .select('id, email, phone, created_at')
    .single();

  if (error) return res.status(500).json({ error: error.message });
  res.json({ profile: data });
});

export default router;
