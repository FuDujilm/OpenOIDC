ALTER TABLE provider_configs ALTER COLUMN provider TYPE VARCHAR(30);
ALTER TABLE provider_configs DROP COLUMN IF EXISTS sort_order;