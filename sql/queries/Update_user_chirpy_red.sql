-- name: UpdateUserIsChirpyRed :exec
UPDATE users
SET is_chirpy_red = $1, updated_at = $2
WHERE id = $3;