-- Content-mode FTS5 table: SQLite reads from torrents when a column is
-- requested, so no duplicate data is stored (only the FTS index).
CREATE VIRTUAL TABLE torrents_fts USING fts5(
    name,
    content=torrents,
    content_rowid=id
);

-- Backfill existing rows.
INSERT INTO torrents_fts (rowid, name)
SELECT id, name FROM torrents;

-- Keep the index in sync with the torrents table.
CREATE TRIGGER torrents_ai AFTER INSERT ON torrents BEGIN
    INSERT INTO torrents_fts (rowid, name) VALUES (new.id, new.name);
END;

CREATE TRIGGER torrents_ad AFTER DELETE ON torrents BEGIN
    INSERT INTO torrents_fts (torrents_fts, rowid, name) VALUES ('delete', old.id, old.name);
END;

CREATE TRIGGER torrents_au AFTER UPDATE ON torrents BEGIN
    INSERT INTO torrents_fts (torrents_fts, rowid, name) VALUES ('delete', old.id, old.name);
    INSERT INTO torrents_fts (rowid, name) VALUES (new.id, new.name);
END;
