CREATE TABLE category_torrents (
    category_id INTEGER NOT NULL REFERENCES categories (id),
    torrent_id  INTEGER NOT NULL REFERENCES torrents (id),
    PRIMARY KEY (category_id, torrent_id)
);

CREATE INDEX index_category_torrents_on_torrent_id_and_category_id ON category_torrents (torrent_id, category_id);
