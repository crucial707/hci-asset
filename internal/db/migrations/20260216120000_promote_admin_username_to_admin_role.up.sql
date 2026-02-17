-- One-off data fix: grant admin role and set password to 'password' for username 'admin'
-- (solves chicken-and-egg when that user was created before roles or as viewer)
-- bcrypt hash below is for the password "password" (use $h$ so $ in hash are not interpreted as dollar-quoting)
UPDATE users
SET role = 'admin', password_hash = $h$$2a$10$UR8yEybXJwD4qAQlvn7RBOWhdIgauhcuS0Rfepa7q.01aT7GT.6kK$h$
WHERE LOWER(TRIM(username)) = 'admin';
