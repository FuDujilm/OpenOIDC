-- Add human-readable numeric user UID while keeping UUID as internal primary key.

CREATE SEQUENCE IF NOT EXISTS users_uid_seq;

ALTER TABLE users ADD COLUMN IF NOT EXISTS uid BIGINT;

WITH numbered AS (
    SELECT
        id,
        (SELECT COALESCE(MAX(uid), 0) FROM users) + ROW_NUMBER() OVER (ORDER BY created_at, id) AS uid
    FROM users
    WHERE uid IS NULL
)
UPDATE users u
SET uid = numbered.uid
FROM numbered
WHERE u.id = numbered.id;

SELECT setval(
    'users_uid_seq',
    GREATEST((SELECT COALESCE(MAX(uid), 0) + 1 FROM users), 1),
    false
);

ALTER TABLE users ALTER COLUMN uid SET DEFAULT nextval('users_uid_seq');
ALTER TABLE users ALTER COLUMN uid SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_uid ON users(uid);