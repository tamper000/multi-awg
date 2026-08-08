-- +goose Up

CREATE TABLE peers_new (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL,
    name       TEXT NOT NULL,
    peer_name  TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (user_id, name)
);

INSERT INTO peers_new (id, user_id, name, peer_name, created_at)
SELECT id, user_id, name, peer_name, created_at
FROM peers;

DROP TABLE peers;
ALTER TABLE peers_new RENAME TO peers;

-- +goose Down

ALTER TABLE peers ADD COLUMN sub_token TEXT NOT NULL DEFAULT '';
