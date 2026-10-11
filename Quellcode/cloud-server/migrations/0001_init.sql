-- Cloud backend schema for SQUAD BASE. Append-only chats unless the user deletes their account.
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS users (
 username TEXT PRIMARY KEY NOT NULL,
 salt TEXT NOT NULL,
 pass_hash TEXT NOT NULL,
 created_at INTEGER NOT NULL,
 last_seen INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS sessions (
 token_hash TEXT PRIMARY KEY NOT NULL,
 username TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_expiry ON sessions(expires_at);
CREATE TABLE IF NOT EXISTS friend_requests (
 sender TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 recipient TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 created_at INTEGER NOT NULL,
 PRIMARY KEY(sender, recipient), CHECK(sender != recipient)
);
CREATE TABLE IF NOT EXISTS friendships (
 user_a TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 user_b TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 PRIMARY KEY(user_a, user_b), CHECK(user_a < user_b)
);
CREATE TABLE IF NOT EXISTS blocks (
 blocker TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 blocked TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 PRIMARY KEY(blocker, blocked), CHECK(blocker != blocked)
);
CREATE TABLE IF NOT EXISTS messages (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 sender TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 recipient TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 body TEXT,
 file_id TEXT,
 file_name TEXT,
 file_size INTEGER,
 created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_message_sender_recipient ON messages(sender, recipient, id);
CREATE INDEX IF NOT EXISTS idx_message_recipient_sender ON messages(recipient, sender, id);
CREATE TABLE IF NOT EXISTS signals (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 sender TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 recipient TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 kind TEXT NOT NULL,
 data_json TEXT,
 created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_signals_recipient ON signals(recipient,id);
CREATE TABLE IF NOT EXISTS files (
 id TEXT PRIMARY KEY NOT NULL,
 sender TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 recipient TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
 file_name TEXT NOT NULL,
 size INTEGER NOT NULL,
 created_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS auth_limits (
 address_hash TEXT PRIMARY KEY NOT NULL,
 attempts INTEGER NOT NULL DEFAULT 0,
 period_start INTEGER NOT NULL,
 blocked_until INTEGER NOT NULL DEFAULT 0
);
