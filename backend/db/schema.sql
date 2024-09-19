-- Users Table
CREATE TABLE users (
    id TEXT PRIMARY KEY UNIQUE,
    username TEXT NOT NULL,
    ip TEXT,
    avatar TEXT,
    public_key TEXT DEFAULT "TEST"
);

-- Chats Table
CREATE TABLE chats (
    id TEXT PRIMARY KEY UNIQUE,
    name TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User Chats Table (many-to-many relationship)
CREATE TABLE user_chats (
    user_id INTEGER,
    chat_id INTEGER,
    role TEXT,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, chat_id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (chat_id) REFERENCES chats(id)
);

-- Messages Table
CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id INTEGER,
    user_id INTEGER,
    content TEXT NOT NULL,
    signature TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chat_id) REFERENCES chats(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);
