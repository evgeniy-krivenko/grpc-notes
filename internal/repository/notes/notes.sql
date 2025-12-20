-- name: CreateNote :one
INSERT INTO notes (user_id, title, content)
VALUES ($1, $2, $3)
RETURNING id, user_id, title, content, created_at, updated_at;

-- name: GetNote :one
SELECT id, user_id, title, content, created_at, updated_at
FROM notes
WHERE id = $1;

-- name: GetNotesByUserID :many
SELECT id, user_id, title, content, created_at, updated_at
FROM notes
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: DeleteNote :exec
DELETE FROM notes WHERE id = $1;
