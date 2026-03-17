CREATE TABLE peers (
    id          INTEGER  PRIMARY KEY AUTOINCREMENT,
    torrent_id  INTEGER  NOT NULL,
    user_id     INTEGER  NOT NULL,
    peer_id     BLOB     NOT NULL,
    ip          TEXT     NOT NULL,
    port        INTEGER  NOT NULL,
    downloaded  INTEGER  NOT NULL DEFAULT 0,
    uploaded    INTEGER  NOT NULL DEFAULT 0,
    left        INTEGER  NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX index_peers_on_torrent_id_and_user_id ON peers (torrent_id, user_id);
CREATE INDEX index_peers_on_torrent_id ON peers (torrent_id);
CREATE INDEX index_peers_on_user_id   ON peers (user_id);
