-- Adds unread message tracking. Existing messages/accounts are NOT changed.
CREATE TABLE IF NOT EXISTS read_receipts (
  reader TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
  peer TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
  last_read_id INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (reader,peer), CHECK(reader != peer)
);
CREATE INDEX IF NOT EXISTS idx_messages_unread ON messages(recipient,sender,id);
