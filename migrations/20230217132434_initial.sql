-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users
(
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    telegram_id BIGINT UNIQUE NOT NULL
);

CREATE INDEX ON users (telegram_id);

CREATE TABLE orders
(
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID references users (id),
    telegram_id BIGINT,
    source      text,
    time        text,
    phone       text,
    destination text,
    created_at  timestamp        default now()
);
CREATE INDEX ON orders (user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
DROP TABLE orders;
-- +goose StatementEnd
