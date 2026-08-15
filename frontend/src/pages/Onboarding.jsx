import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../lib/api';

const SUGGESTED_TICKERS = ['AAPL', 'TSLA', 'NVDA', 'MSFT', 'AMZN', 'META', 'GOOGL', 'INTC'];

export default function Onboarding() {
  const navigate = useNavigate();
  const [step, setStep] = useState(1);

  const [phone, setPhone] = useState('');
  const [phoneError, setPhoneError] = useState(null);
  const [savingPhone, setSavingPhone] = useState(false);

  const [selected, setSelected] = useState(new Set());
  const [customTicker, setCustomTicker] = useState('');
  const [watchlistError, setWatchlistError] = useState(null);
  const [finishing, setFinishing] = useState(false);

  function toggleTicker(ticker) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(ticker)) {
        next.delete(ticker);
      } else {
        next.add(ticker);
      }
      return next;
    });
  }

  async function handlePhoneSubmit(e) {
    e.preventDefault();
    setPhoneError(null);
    setSavingPhone(true);
    try {
      if (phone.trim()) {
        await api.setPhone(phone.trim());
      }
      setStep(2);
    } catch (err) {
      setPhoneError(err.message);
    } finally {
      setSavingPhone(false);
    }
  }

  function addCustomTicker() {
    const t = customTicker.trim().toUpperCase();
    if (!t) return;
    setSelected((prev) => new Set(prev).add(t));
    setCustomTicker('');
  }

  async function handleFinish() {
    setWatchlistError(null);
    setFinishing(true);
    try {
      await Promise.all([...selected].map((ticker) => api.addTicker(ticker)));
      navigate('/dashboard');
    } catch (err) {
      setWatchlistError(err.message);
    } finally {
      setFinishing(false);
    }
  }

  return (
    <div className="auth-page">
      <div className="auth-card wide">
        <div className="steps-indicator">
          <span className={step >= 1 ? 'active' : ''}>1. Phone</span>
          <span className={step >= 2 ? 'active' : ''}>2. Watchlist</span>
        </div>

        {step === 1 && (
          <form onSubmit={handlePhoneSubmit}>
            <h1>Where should we text you?</h1>
            <p className="subtle">
              Optional for now — you can add or change this anytime in Settings. SMS delivery is off by default.
            </p>
            <label>
              Phone number
              <input
                type="tel"
                placeholder="+15555550123"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
              />
            </label>
            {phoneError && <p className="form-error">{phoneError}</p>}
            <button type="submit" className="btn-primary" disabled={savingPhone}>
              {savingPhone ? 'Saving…' : 'Continue'}
            </button>
            <button type="button" className="btn-link" onClick={() => setStep(2)}>
              Skip for now
            </button>
          </form>
        )}

        {step === 2 && (
          <div>
            <h1>Pick tickers to watch</h1>
            <p className="subtle">You'll see buy/sell/hold signals for these on your dashboard.</p>

            <div className="ticker-grid">
              {SUGGESTED_TICKERS.map((ticker) => (
                <button
                  key={ticker}
                  type="button"
                  className={`ticker-chip ${selected.has(ticker) ? 'selected' : ''}`}
                  onClick={() => toggleTicker(ticker)}
                >
                  {ticker}
                </button>
              ))}
            </div>

            <div className="inline-form">
              <input
                type="text"
                placeholder="Add another ticker, e.g. NFLX"
                value={customTicker}
                onChange={(e) => setCustomTicker(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addCustomTicker())}
              />
              <button type="button" className="btn-ghost" onClick={addCustomTicker}>
                Add
              </button>
            </div>

            {watchlistError && <p className="form-error">{watchlistError}</p>}

            <button type="button" className="btn-primary" onClick={handleFinish} disabled={finishing}>
              {finishing ? 'Saving…' : `Finish (${selected.size} selected)`}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
