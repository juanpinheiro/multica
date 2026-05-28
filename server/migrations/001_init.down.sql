-- Drop all schema objects. Public-schema-only; tables drop with their indexes,
-- constraints, sequences, and triggers via CASCADE. Functions and extensions
-- drop separately. Order doesn't matter under CASCADE.
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
