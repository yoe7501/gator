-- name: CreateFeedFollow :one
WITH inserted AS (
    INSERT INTO feed_follows(id, created_at, updated_at, user_id, feed_id)
    VALUES ($1,$2,$3,$4,$5)
    RETURNING *
)
SELECT
    inserted.*,
    users.name AS user_name,
    feeds.name AS feed_name
FROM inserted
INNER JOIN users ON users.id = inserted.user_id
INNER JOIN feeds ON feeds.id = inserted.feed_id;

-- name: GetFeedFollowsForUser :many
SELECT
    feeds.name AS feed_name
FROM feeds
INNER JOIN feed_follows ON feed_follows.feed_id = feeds.id
WHERE feed_follows.user_id = $1;

-- name: DeleteFollow :exec
DELETE FROM feed_follows
WHERE feed_follows.user_id = $1
AND feed_follows.feed_id IN (SELECT feeds.id FROM feeds WHERE feeds.url = $2);
