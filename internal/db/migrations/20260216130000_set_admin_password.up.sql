-- Set password to 'password' for username 'admin' (bcrypt hash).
-- Use $h$ so $ in the hash are not interpreted as PostgreSQL dollar-quoting.
UPDATE users
SET password_hash = $h$$2a$10$UR8yEybXJwD4qAQlvn7RBOWhdIgauhcuS0Rfepa7q.01aT7GT.6kK$h$
WHERE LOWER(TRIM(username)) = 'admin';
