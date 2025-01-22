CREATE TABLE IF NOT EXISTS requests (
  id CHAR(26) PRIMARY KEY,
  method TEXT,
  path TEXT,
  headers JSONB,
  cookies JSONB,
  body BYTEA,
  query_params JSONB,
  recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS requests_method_path_headers_idx ON requests (method, path, headers);
