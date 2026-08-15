import { useEffect, useState } from 'react';
import { api } from '../lib/api';

const ALL_SIGNAL_TYPES = ['buy', 'sell', 'hold'];

export default function Settings() {
  const [watchlist, setWatchlist] = useState([]);
  const [newTicker, setNewTicker] = useState('');

  const [smsEnabled, setSmsEnabled] = useState(false);
  const [signalTypes, setSignalTypes] = useState(new Set(ALL_SIGNAL_TYPES));

  const [phone, setPhone] = useState('');

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const [{ watchlist: wl }, { preferences }, { profile }] = await Promise.all([
          api.getWatchlist(),
          api.getAlertPreferences(),
          api.getProfile(),
        ]);
        setWatchlist(wl);
        setSmsEnabled(preferences.sms_enabled);
        setSignalTypes(new Set(preferences.signal_types));
        setPhone(profile.phone || '');
      } catch (err) {
        setError(err.message);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  async function handleAddTicker(e) {
    e.preventDefault();
    const ticker = newTicker.trim().toUpperCase();
    if (!ticker) return;
    try {
      const { watchlist: row } = await api.addTicker(ticker);
      setWatchlist((prev) => [row, ...prev.filter((w) => w.ticker !== ticker)]);
      setNewTicker('');
    } catch (err) {
      setError(err.message);
    }
  }

  async function handleRemoveTicker(ticker) {
    try {
      await api.removeTicker(ticker);
      setWatchlist((prev) => prev.filter((w) => w.ticker !== ticker));
    } catch (err) {
      setError(err.message);
    }
  }

  function toggleSignalType(type) {
    setSignalTypes((prev) => {
      const next = new Set(prev);
      if (next.has(type)) {
        next.delete(type);
      } else {
        next.add(type);
      }
      return next;
    });
  }

  async function savePreferences() {
    setError(null);
    setSaved(false);
    try {
      await api.setAlertPreferences({ sms_enabled: smsEnabled, signal_types: [...signalTypes] });
      setSaved(true);
    } catch (err) {
      setError(err.message);
    }
  }

  async function savePhone(e) {
    e.preventDefault();
    setError(null);
    try {
      await api.setPhone(phone.trim());
      setSaved(true);
    } catch (err) {
      setError(err.message);
    }
  }

  if (loading) return <div className="page"><p className="subtle">Loading…</p></div>;

  return (
    <div className="page">
      <h1>Settings</h1>
      {error && <p className="form-error">{error}</p>}
      {saved && <p className="form-info">Saved.</p>}

      <section className="settings-section">
        <h2>SMS alerts</h2>
        <label className="toggle-row">
          <input type="checkbox" checked={smsEnabled} onChange={(e) => setSmsEnabled(e.target.checked)} />
          {smsEnabled ? 'Alerts are on' : 'Alerts are paused'}
        </label>

        <div className="signal-type-row">
          {ALL_SIGNAL_TYPES.map((type) => (
            <label key={type} className="checkbox-pill">
              <input type="checkbox" checked={signalTypes.has(type)} onChange={() => toggleSignalType(type)} />
              {type[0].toUpperCase() + type.slice(1)}
            </label>
          ))}
        </div>

        <button type="button" className="btn-primary" onClick={savePreferences}>
          Save preferences
        </button>
      </section>

      <section className="settings-section">
        <h2>Phone number</h2>
        <form className="inline-form" onSubmit={savePhone}>
          <input type="tel" placeholder="+15555550123" value={phone} onChange={(e) => setPhone(e.target.value)} />
          <button type="submit" className="btn-ghost">
            Save
          </button>
        </form>
      </section>

      <section className="settings-section">
        <h2>Watchlist</h2>
        <form className="inline-form" onSubmit={handleAddTicker}>
          <input
            type="text"
            placeholder="Add a ticker, e.g. AAPL"
            value={newTicker}
            onChange={(e) => setNewTicker(e.target.value)}
          />
          <button type="submit" className="btn-ghost">
            Add
          </button>
        </form>

        <ul className="watchlist">
          {watchlist.map((w) => (
            <li key={w.id}>
              <span>{w.ticker}</span>
              <button type="button" className="btn-link" onClick={() => handleRemoveTicker(w.ticker)}>
                Remove
              </button>
            </li>
          ))}
          {watchlist.length === 0 && <p className="subtle">No tickers yet.</p>}
        </ul>
      </section>
    </div>
  );
}
