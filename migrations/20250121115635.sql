CREATE TABLE IF NOT EXISTS requests (
  id CHAR(26) PRIMARY KEY,
  method TEXT NOT NULL,
  path TEXT NOT NULL,
  headers JSONB NULL,
  cookies JSONB NULL,
  body BYTEA NULL,
  query_params JSONB NULL,
  recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS requests_method_path_idx ON requests (method, path);
