-- name: UpsertResource :exec
INSERT INTO resources (provider, account_id, service, resource_id, resource_type, name, region, spec, tags, state, discovered_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider, account_id, resource_id)
DO UPDATE SET
    service = excluded.service,
    resource_type = excluded.resource_type,
    name = excluded.name,
    region = excluded.region,
    spec = excluded.spec,
    tags = excluded.tags,
    state = excluded.state,
    discovered_at = excluded.discovered_at;

-- name: GetResourcesByProvider :many
SELECT * FROM resources
WHERE provider = ?
ORDER BY service, resource_type;

-- name: GetResourcesByAccount :many
SELECT * FROM resources
WHERE provider = ? AND account_id = ?
ORDER BY service, resource_type;

-- name: GetResourcesByService :many
SELECT * FROM resources
WHERE provider = ? AND service = ?
ORDER BY resource_type, name;

-- name: GetResourcesByServiceAndRegion :many
SELECT * FROM resources
WHERE provider = ? AND service = ? AND region = ?
ORDER BY resource_type, name;

-- name: GetResourcesByRegion :many
SELECT * FROM resources
WHERE provider = ? AND region = ?
ORDER BY service, resource_type;

-- name: CountResources :one
SELECT COUNT(*) FROM resources;

-- name: CountResourcesByProvider :one
SELECT COUNT(*) FROM resources WHERE provider = ?;

-- name: CountResourcesByService :many
SELECT service, COUNT(*) AS count FROM resources
WHERE provider = ?
GROUP BY service
ORDER BY count DESC;

-- name: CountSpotInstances :one
SELECT COUNT(*) FROM resources
WHERE provider = ? AND resource_type = 'ec2:instance' AND spec LIKE '%"lifecycle":"spot"%';

-- name: CountResourcesByAccount :many
SELECT account_id, COUNT(*) AS count FROM resources
WHERE provider = ?
GROUP BY account_id
ORDER BY count DESC;

-- name: DeleteResourcesByProvider :exec
DELETE FROM resources WHERE provider = ?;

-- name: DeleteResourcesByAccount :exec
DELETE FROM resources WHERE provider = ? AND account_id = ?;
