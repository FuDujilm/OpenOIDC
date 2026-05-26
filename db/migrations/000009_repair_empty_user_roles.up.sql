UPDATE users SET role = 'user' WHERE role IS NULL OR BTRIM(role) = '';
