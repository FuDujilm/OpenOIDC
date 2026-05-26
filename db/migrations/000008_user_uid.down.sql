-- Remove human-readable numeric user UID.

DROP INDEX IF EXISTS idx_users_uid;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'users'
          AND column_name = 'uid'
    ) THEN
        ALTER TABLE users ALTER COLUMN uid DROP DEFAULT;
        ALTER TABLE users DROP COLUMN uid;
    END IF;
END $$;

DROP SEQUENCE IF EXISTS users_uid_seq;