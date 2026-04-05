-- +goose up
CREATE TABLE users (
    id UUID NOT NULL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    email TEXT NOT NULL UNIQUE
);

-- +goose down
DROP TABLE users;