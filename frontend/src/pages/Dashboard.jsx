import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from '../lib/api';
import { supabase } from '../lib/supabaseClient';
import SignalCard from '../components/SignalCard';

const SIGNAL_FILTERS = ['all', 'buy', 'sell', 'hold'];

export default function Dashboard() {
  const [signals, setSignals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [tickerFilter, setTickerFilter] = useState('');
  const [signalFilter, setSignalFilter] = useState('all');
  const refreshTimer = useRef(null);

  const fetchSignals = useCallback(async () => {
    try {
      const params = {};
      if (tickerFilter.trim()) params.ticker = tickerFilter.trim();
      if (signalFilter !== 'all') params.signal = signalFilter;
      const { signals: rows } = await api.getSignals(params);
      setSignals(rows);
      setError(null);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [tickerFilter, signalFilter]);

  useEffect(() => {
    setLoading(true);
    fetchSignals();
  }, [fetchSignals]);

  // Live feed: new tweet_analysis rows stream in via Supabase Realtime.
  // Debounced refetch keeps the joined tweet/source data consistent instead
  // of hand-merging a partial insert payload.
  useEffect(() => {
    const channel = supabase
      .channel('tweet_analysis-live')
      .on('postgres_changes', { event: 'INSERT', schema: 'public', table: 'tweet_analysis' }, () => {
        clearTimeout(refreshTimer.current);
        refreshTimer.current = setTimeout(fetchSignals, 400);
      })
      .subscribe();

    return () => {
      clearTimeout(refreshTimer.current);
      supabase.removeChannel(channel);
    };
  }, [fetchSignals]);

  return (
    <div className="page">
      <div className="page-header">
        <h1>Live signals</h1>
        <div className="filters">
          <input
            type="text"
            placeholder="Filter by ticker (e.g. AAPL)"
            value={tickerFilter}
            onChange={(e) => setTickerFilter(e.target.value)}
          />
          <select value={signalFilter} onChange={(e) => setSignalFilter(e.target.value)}>
            {SIGNAL_FILTERS.map((s) => (
              <option key={s} value={s}>
                {s === 'all' ? 'All signals' : s[0].toUpperCase() + s.slice(1)}
              </option>
            ))}
          </select>
        </div>
      </div>

      {error && <p className="form-error">{error}</p>}
      {loading && <p className="subtle">Loading signals…</p>}
      {!loading && signals.length === 0 && !error && (
        <p className="subtle">No signals yet. Once the scraper and analysis pipeline are running, they'll show up here in real time.</p>
      )}

      <div className="signal-list">
        {signals.map((s) => (
          <SignalCard key={s.id} signal={s} />
        ))}
      </div>
    </div>
  );
}
