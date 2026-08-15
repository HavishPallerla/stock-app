import 'dotenv/config';
import express from 'express';
import cors from 'cors';
import helmet from 'helmet';
import morgan from 'morgan';

import watchlistRouter from './routes/watchlist.js';
import alertPreferencesRouter from './routes/alertPreferences.js';
import signalsRouter from './routes/signals.js';
import profileRouter from './routes/profile.js';

const app = express();
const port = process.env.PORT || 4000;
const allowedOrigin = process.env.FRONTEND_ORIGIN || 'http://localhost:5173';

app.use(helmet());
app.use(cors({ origin: allowedOrigin }));
app.use(express.json());
app.use(morgan('dev'));

app.get('/api/health', (_req, res) => res.json({ status: 'ok' }));

app.use('/api/watchlist', watchlistRouter);
app.use('/api/alert-preferences', alertPreferencesRouter);
app.use('/api/signals', signalsRouter);
app.use('/api/profile', profileRouter);

app.use((err, _req, res, _next) => {
  console.error(err);
  res.status(500).json({ error: 'Internal server error' });
});

app.listen(port, () => {
  console.log(`api-express listening on :${port}`);
});
