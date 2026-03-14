-- Change default for new subscribers to unconfirmed (requires email confirmation).
-- Existing confirmed subscribers are left as-is.
ALTER TABLE subscribers ALTER COLUMN confirmed SET DEFAULT false;
