import { supabase } from './supabaseClient';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:4000';

async function authHeader() {
  const {
    data: { session },
  } = await supabase.auth.getSession();
  return session ? { Authorization: `Bearer ${session.access_token}` } : {};
}

async function request(path, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...(await authHeader()),
    ...options.headers,
  };

  const res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers });

  if (res.status === 204) return null;

  const body = await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error(body?.error || `Request failed: ${res.status}`);
  }
  return body;
}

export const api = {
  getProfile: () => request('/api/profile'),
  setPhone: (phone) => request('/api/profile/phone', { method: 'PUT', body: JSON.stringify({ phone }) }),

  getWatchlist: () => request('/api/watchlist'),
  addTicker: (ticker) => request('/api/watchlist', { method: 'POST', body: JSON.stringify({ ticker }) }),
  removeTicker: (ticker) => request(`/api/watchlist/${encodeURIComponent(ticker)}`, { method: 'DELETE' }),

  getAlertPreferences: () => request('/api/alert-preferences'),
  setAlertPreferences: (prefs) =>
    request('/api/alert-preferences', { method: 'PUT', body: JSON.stringify(prefs) }),

  getSignals: (params = {}) => {
    const qs = new URLSearchParams(params).toString();
    return request(`/api/signals${qs ? `?${qs}` : ''}`);
  },
};
