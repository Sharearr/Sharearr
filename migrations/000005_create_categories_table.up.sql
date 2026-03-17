CREATE TABLE categories (
    id         INTEGER  PRIMARY KEY,
    name       TEXT     NOT NULL,
    parent_id  INTEGER  REFERENCES categories (id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX index_categories_on_parent_id ON categories (parent_id);

INSERT INTO categories (id, name) VALUES
    (1000, 'Console'),
    (2000, 'Movies'),
    (3000, 'Audio'),
    (4000, 'PC'),
    (5000, 'TV'),
    (6000, 'XXX'),
    (7000, 'Books'),
    (8000, 'Other');

-- Console
INSERT INTO categories (id, name, parent_id) VALUES
    (1010, 'NDS',            1000),
    (1020, 'PSP',            1000),
    (1030, 'Wii',            1000),
    (1040, 'Xbox',           1000),
    (1050, 'Xbox 360',       1000),
    (1060, 'Wiiware',        1000),
    (1070, 'Xbox 360 DLC',   1000),
    (1080, 'PS3',            1000),
    (1090, 'Other',          1000),
    (1110, '3DS',            1000),
    (1120, 'PS Vita',        1000),
    (1130, 'WiiU',           1000),
    (1140, 'Xbox One',       1000),
    (1150, 'PS4',            1000),
    (1160, 'Switch',         1000),
    (1170, 'PS5',            1000),
    (1180, 'Xbox Series X',  1000);

-- Movies
INSERT INTO categories (id, name, parent_id) VALUES
    (2010, 'Foreign',        2000),
    (2020, 'Other',          2000),
    (2030, 'SD',             2000),
    (2040, 'HD',             2000),
    (2045, 'UHD',            2000),
    (2050, 'BluRay',         2000),
    (2060, '3D',             2000),
    (2070, 'DVD',            2000),
    (2080, 'WEB-DL',         2000);

-- Audio
INSERT INTO categories (id, name, parent_id) VALUES
    (3010, 'MP3',            3000),
    (3020, 'Video',          3000),
    (3030, 'Audiobook',      3000),
    (3040, 'Lossless',       3000),
    (3050, 'Podcast',        3000),
    (3060, 'Other',          3000);

-- PC
INSERT INTO categories (id, name, parent_id) VALUES
    (4010, '0day',             4000),
    (4020, 'ISO',              4000),
    (4030, 'Mac',              4000),
    (4040, 'Mobile (Other)',   4000),
    (4050, 'Games',            4000),
    (4060, 'Mobile (iOS)',     4000),
    (4070, 'Mobile (Android)', 4000);

-- TV
INSERT INTO categories (id, name, parent_id) VALUES
    (5010, 'WEB-DL',         5000),
    (5020, 'Foreign',        5000),
    (5030, 'SD',             5000),
    (5040, 'HD',             5000),
    (5045, 'UHD',            5000),
    (5050, 'Other',          5000),
    (5060, 'Sport',          5000),
    (5070, 'Anime',          5000),
    (5080, 'Documentary',    5000);

-- XXX
INSERT INTO categories (id, name, parent_id) VALUES
    (6010, 'DVD',            6000),
    (6020, 'WMV',            6000),
    (6030, 'XviD',           6000),
    (6040, 'x264',           6000),
    (6050, 'UHD',            6000),
    (6060, 'Pack',           6000),
    (6070, 'ImageSet',       6000),
    (6080, 'Other',          6000);

-- Books
INSERT INTO categories (id, name, parent_id) VALUES
    (7010, 'Mags',           7000),
    (7020, 'EBook',          7000),
    (7030, 'Comics',         7000),
    (7040, 'Technical',      7000),
    (7050, 'Other',          7000),
    (7060, 'Foreign',        7000);

-- Other
INSERT INTO categories (id, name, parent_id) VALUES
    (8010, 'Misc',           8000),
    (8020, 'Hashed',         8000);
