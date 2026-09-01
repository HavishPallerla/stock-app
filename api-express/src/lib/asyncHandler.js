// Express 4 does not forward a rejected promise from an async route handler
// to the error-handling middleware — an uncaught throw just hangs the
// request instead of producing a 500. Wrap every async handler with this so
// unexpected errors (not the routes' own `{ data, error }` checks, but real
// exceptions) reach app.use((err, ...) => ...) in index.js.
export function asyncHandler(fn) {
  return (req, res, next) => {
    Promise.resolve(fn(req, res, next)).catch(next);
  };
}
