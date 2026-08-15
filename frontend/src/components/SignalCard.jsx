const SIGNAL_LABEL = { buy: 'Buy', sell: 'Sell', hold: 'Hold' };

function timeAgo(iso) {
  if (!iso) return '';
  const diffMs = Date.now() - new Date(iso).getTime();
  const mins = Math.round(diffMs / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.round(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.round(hours / 24)}d ago`;
}

export default function SignalCard({ signal }) {
  const tweet = signal.tweets;
  const title = tweet?.title || tweet?.content;
  const source = tweet?.source_account;

  return (
    <article className={`signal-card signal-${signal.signal}`}>
      <div className="signal-card-top">
        <span className="ticker">${signal.ticker}</span>
        <span className={`badge badge-${signal.signal}`}>{SIGNAL_LABEL[signal.signal] ?? signal.signal}</span>
        <span className="confidence">{Math.round(signal.confidence * 100)}% confidence</span>
      </div>

      {title && (
        tweet?.url ? (
          <a className="signal-headline" href={tweet.url} target="_blank" rel="noreferrer">
            {title}
          </a>
        ) : (
          <p className="signal-headline">{title}</p>
        )
      )}

      <p className="justification">{signal.justification}</p>

      <div className="signal-card-meta">
        {source && <span>{source}</span>}
        <span>{timeAgo(signal.created_at)}</span>
      </div>
    </article>
  );
}
