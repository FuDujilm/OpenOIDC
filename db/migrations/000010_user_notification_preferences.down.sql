-- Remove notification preferences from users table
ALTER TABLE users DROP COLUMN IF EXISTS risk_report_email_enabled;
