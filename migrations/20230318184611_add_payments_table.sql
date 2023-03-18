-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS moddatetime;

CREATE TABLE payments
(
    id               uuid,
    order_id         BIGINT references orders (id),
    status           text,
    confirmation_url text,
    updated_at       timestamptz default now(),
    created_at       timestamptz default now()
);
CREATE INDEX ON payments (order_id);

CREATE TRIGGER mdt_payments
    BEFORE UPDATE ON payments
    FOR EACH ROW
EXECUTE PROCEDURE moddatetime (updated_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE payments;
-- +goose StatementEnd
