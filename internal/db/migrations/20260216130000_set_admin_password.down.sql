-- Revert: clear password for username 'admin'
UPDATE users SET password_hash = NULL WHERE LOWER(TRIM(username)) = 'admin';
