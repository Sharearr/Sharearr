CREATE TABLE torrents (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    info_hash  TEXT     NOT NULL,
    name       TEXT     NOT NULL,
    size_bytes INTEGER  NOT NULL,
    file       BLOB     NOT NULL,
    user_id    INTEGER  NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX index_torrents_on_info_hash ON torrents (info_hash);
CREATE INDEX index_torrents_on_user_id ON torrents (user_id);
