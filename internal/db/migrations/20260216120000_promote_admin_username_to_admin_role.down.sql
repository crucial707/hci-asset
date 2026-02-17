-- Revert: set role back to viewer and clear password for username 'admin' (best-effort)
UPDATE users SET role = 'viewer', password_hash = NULL WHERE LOWER(TRIM(username)) = 'admin';
