-- +goose Up
-- +goose StatementBegin
create table if not exists notes (
    id         bigserial primary key,
    user_id    bigint      not null,
    title      varchar     not null,
    content    text        not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index idx_notes_user_id on notes(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists notes;
-- +goose StatementEnd
