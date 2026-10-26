CREATE TABLE admins (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 username VARCHAR(64) NOT NULL,
 display_name VARCHAR(100) NOT NULL DEFAULT '',
 password_hash VARCHAR(255) NOT NULL,
 enabled BOOLEAN NOT NULL DEFAULT TRUE,
 created_at DATETIME NOT NULL,
 updated_at DATETIME NOT NULL,
 UNIQUE KEY uk_admins_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE admin_login_logs (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 admin_id BIGINT UNSIGNED NULL,
 username VARCHAR(64) NOT NULL,
 success BOOLEAN NOT NULL,
 reason VARCHAR(64) NOT NULL,
 ip VARCHAR(64) NOT NULL,
 user_agent VARCHAR(512) NOT NULL,
 created_at DATETIME NOT NULL,
 KEY idx_login_admin_created (admin_id, created_at),
 KEY idx_login_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE settings (
 `key` VARCHAR(128) NOT NULL PRIMARY KEY,
 `value` JSON NOT NULL,
 created_at DATETIME NOT NULL,
 updated_at DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
