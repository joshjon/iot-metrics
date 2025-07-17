CREATE TABLE configs
(
    device_id             TEXT PRIMARY KEY,
    temperature_threshold REAL    NOT NULL,
    battery_threshold     INTEGER NOT NULL
);

CREATE TABLE metrics
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id   TEXT    NOT NULL,
    temperature REAL    NOT NULL,
    battery     INTEGER NOT NULL,
    timestamp   INTEGER NOT NULL -- unix
);

CREATE TABLE alerts
(
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT    NOT NULL,
    reason    TEXT    NOT NULL,
    desc      TEXT    NOT NULL,
    timestamp INTEGER NOT NULL -- unix
);
