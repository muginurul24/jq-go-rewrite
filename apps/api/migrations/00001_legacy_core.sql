-- +goose Up
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    email_verified_at TIMESTAMP NULL,
    password VARCHAR(255) NOT NULL,
    remember_token VARCHAR(100) NULL,
    app_authentication_secret TEXT NULL,
    app_authentication_recovery_codes TEXT NULL,
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('dev', 'superadmin', 'admin', 'user')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE password_reset_tokens (
    email VARCHAR(255) PRIMARY KEY,
    token VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NULL
);

CREATE TABLE sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    ip_address VARCHAR(45) NULL,
    user_agent TEXT NULL,
    payload TEXT NOT NULL,
    last_activity INTEGER NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_last_activity_idx ON sessions (last_activity);

CREATE TABLE personal_access_tokens (
    id BIGSERIAL PRIMARY KEY,
    tokenable_type VARCHAR(255) NOT NULL,
    tokenable_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    token VARCHAR(64) NOT NULL UNIQUE,
    abilities TEXT NULL,
    last_used_at TIMESTAMP NULL,
    expires_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX personal_access_tokens_tokenable_idx
    ON personal_access_tokens (tokenable_type, tokenable_id);
CREATE INDEX personal_access_tokens_expires_at_idx
    ON personal_access_tokens (expires_at);

CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    type VARCHAR(255) NOT NULL,
    notifiable_type VARCHAR(255) NOT NULL,
    notifiable_id BIGINT NOT NULL,
    data JSONB NOT NULL,
    read_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX notifications_notifiable_idx
    ON notifications (notifiable_type, notifiable_id);

CREATE TABLE tokos (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    callback_url TEXT NULL UNIQUE,
    token VARCHAR(255) NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX tokos_user_id_idx ON tokos (user_id);

CREATE TABLE banks (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bank_code TEXT NOT NULL CHECK (bank_code IN ('002', '008', '009', '014', '501', '022', '013', '111', '451', '542', '490')),
    bank_name TEXT NOT NULL CHECK (bank_name IN ('BRI', 'Mandiri', 'BNI', 'BCA', 'Blu BCA Digital', 'CIMB', 'Permata', 'DKI', 'BSI', 'JAGO', 'NEO')),
    account_number VARCHAR(255) NOT NULL UNIQUE,
    account_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX banks_user_id_idx ON banks (user_id);

CREATE TABLE balances (
    id BIGSERIAL PRIMARY KEY,
    toko_id BIGINT NOT NULL REFERENCES tokos(id) ON DELETE CASCADE,
    settle NUMERIC(10, 0) NOT NULL DEFAULT 0,
    pending NUMERIC(10, 0) NOT NULL DEFAULT 0,
    nexusggr NUMERIC(10, 0) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX balances_toko_id_idx ON balances (toko_id);

CREATE TABLE incomes (
    id BIGSERIAL PRIMARY KEY,
    ggr INTEGER NOT NULL DEFAULT 7,
    fee_transaction INTEGER NOT NULL DEFAULT 3,
    fee_withdrawal INTEGER NOT NULL DEFAULT 15,
    amount NUMERIC(10, 0) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    toko_id BIGINT NOT NULL REFERENCES tokos(id) ON DELETE CASCADE,
    player VARCHAR(255) NULL,
    external_player VARCHAR(255) NULL,
    category TEXT NOT NULL CHECK (category IN ('qris', 'nexusggr')),
    type TEXT NOT NULL CHECK (type IN ('deposit', 'withdrawal')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'success', 'failed', 'expired')),
    amount NUMERIC(10, 0) NOT NULL,
    code VARCHAR(255) NULL,
    note TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX transactions_toko_id_idx ON transactions (toko_id);
CREATE INDEX transactions_external_player_idx ON transactions (external_player);
CREATE INDEX transactions_code_idx ON transactions (code);

CREATE TABLE players (
    id BIGSERIAL PRIMARY KEY,
    toko_id BIGINT NOT NULL REFERENCES tokos(id) ON DELETE CASCADE,
    username VARCHAR(255) NOT NULL,
    ext_username VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX players_toko_id_idx ON players (toko_id);
CREATE INDEX players_ext_username_idx ON players (ext_username);

-- +goose Down
DROP TABLE IF EXISTS players;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS incomes;
DROP TABLE IF EXISTS balances;
DROP TABLE IF EXISTS banks;
DROP TABLE IF EXISTS tokos;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS personal_access_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS users;
