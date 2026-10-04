-- ─────────────────────────────────────────────────────────────
-- Schema PostgreSQL — Family Finance App
-- ─────────────────────────────────────────────────────────────

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── families ─────────────────────────────────────────────────
-- Satu baris per keluarga; semua data domain terikat ke family_id
CREATE TABLE IF NOT EXISTS families (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  name       VARCHAR(100) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── users ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id     UUID         NOT NULL REFERENCES families(id) ON DELETE CASCADE,
  username      VARCHAR(50)  NOT NULL UNIQUE,
  email         VARCHAR(100) NOT NULL UNIQUE,
  password_hash TEXT         NOT NULL,
  role          VARCHAR(10)  NOT NULL DEFAULT 'member' CHECK (role IN ('owner','member')),
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_family_id ON users(family_id);

-- ── sessions ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sessions (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '7 days'
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

-- ── categories ────────────────────────────────────────────────
-- Kategori kustom per keluarga; nama unik dalam satu family
CREATE TABLE IF NOT EXISTS categories (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id  UUID        NOT NULL REFERENCES families(id) ON DELETE CASCADE,
  name       VARCHAR(80) NOT NULL,
  type       VARCHAR(10) NOT NULL DEFAULT 'both' CHECK (type IN ('income','expense','both')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (family_id, name)
);

CREATE INDEX IF NOT EXISTS idx_categories_family_id ON categories(family_id);

-- ── wallets ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS wallets (
  id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id  UUID          NOT NULL REFERENCES families(id) ON DELETE CASCADE,
  name       VARCHAR(100)  NOT NULL,
  type       VARCHAR(20)   NOT NULL DEFAULT 'cash' CHECK (type IN ('cash','bank','e-wallet','investment','other')),
  balance    NUMERIC(15,2) NOT NULL DEFAULT 0,
  color      VARCHAR(20)   NOT NULL DEFAULT '#6366f1',
  note       TEXT          NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  UNIQUE (family_id, name)
);

CREATE INDEX IF NOT EXISTS idx_wallets_family_id ON wallets(family_id);

-- ── wallet_transfers ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS wallet_transfers (
  id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id      UUID          NOT NULL REFERENCES families(id) ON DELETE CASCADE,
  from_wallet_id UUID          NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  to_wallet_id   UUID          NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
  amount         NUMERIC(15,2) NOT NULL CHECK (amount > 0),
  note           TEXT          NOT NULL DEFAULT '',
  date           DATE          NOT NULL,
  created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wallet_transfers_family_id ON wallet_transfers(family_id);

-- ── transactions ──────────────────────────────────────────────
-- debt_payment_id FK ditambahkan via ALTER TABLE di bawah
-- setelah tabel debt_payments ada (circular reference)
CREATE TABLE IF NOT EXISTS transactions (
  id                UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id         UUID          NOT NULL REFERENCES families(id) ON DELETE CASCADE,
  created_by        UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type              VARCHAR(10)   NOT NULL CHECK (type IN ('income','expense')),
  amount            NUMERIC(15,2) NOT NULL CHECK (amount > 0),
  category_name     VARCHAR(80)   NOT NULL,
  member            VARCHAR(50)   NOT NULL,
  date              DATE          NOT NULL,
  note              TEXT          NOT NULL DEFAULT '',
  debt_payment_id   UUID          NULL,   -- FK ditambahkan via ALTER TABLE di bawah
  saving_deposit_id UUID          NULL,   -- FK ditambahkan via ALTER TABLE di bawah
  wallet_id         UUID          NULL    REFERENCES wallets(id) ON DELETE SET NULL,
  created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transactions_family_id ON transactions(family_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date      ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_type      ON transactions(type);

-- ── budgets ───────────────────────────────────────────────────
-- Anggaran per kategori per bulan; UNIQUE agar upsert aman
CREATE TABLE IF NOT EXISTS budgets (
  id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id     UUID          NOT NULL REFERENCES families(id) ON DELETE CASCADE,
  category_name VARCHAR(80)   NOT NULL,
  month         CHAR(7)       NOT NULL,   -- YYYY-MM
  amount        NUMERIC(15,2) NOT NULL CHECK (amount >= 0),
  created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  UNIQUE (family_id, category_name, month)
);

CREATE INDEX IF NOT EXISTS idx_budgets_family_id ON budgets(family_id);
CREATE INDEX IF NOT EXISTS idx_budgets_month     ON budgets(month);

-- ── emergency_fund ────────────────────────────────────────────
-- Tepat satu baris per family; target = monthly_expense_avg * target_months
CREATE TABLE IF NOT EXISTS emergency_fund (
  id                  UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id           UUID          NOT NULL UNIQUE REFERENCES families(id) ON DELETE CASCADE,
  current_balance     NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (current_balance >= 0),
  monthly_expense_avg NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (monthly_expense_avg >= 0),
  target_balance      NUMERIC(15,2) NOT NULL DEFAULT 0,  -- stored: monthly_expense_avg * target_months
  months_covered      NUMERIC(6,2)  NOT NULL DEFAULT 0,  -- stored: current_balance / monthly_expense_avg
  target_months       INT           NOT NULL DEFAULT 6 CHECK (target_months > 0),  -- default 6 bulan
  updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- ── emergency_fund_deposits ───────────────────────────────────
-- Riwayat setoran (+) dan penarikan (-) dana darurat
CREATE TABLE IF NOT EXISTS emergency_fund_deposits (
  id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id  UUID          NOT NULL REFERENCES families(id) ON DELETE CASCADE,
  amount     NUMERIC(15,2) NOT NULL,  -- positif = setor, negatif = tarik
  note       TEXT          NOT NULL DEFAULT '',
  date       DATE          NOT NULL,
  created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ef_deposits_family_id ON emergency_fund_deposits(family_id);
CREATE INDEX IF NOT EXISTS idx_ef_deposits_date      ON emergency_fund_deposits(date);

-- ── debts ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS debts (
  id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id        UUID          NOT NULL REFERENCES families(id) ON DELETE CASCADE,
  name             VARCHAR(150)  NOT NULL,
  total_amount     NUMERIC(15,2) NOT NULL CHECK (total_amount > 0),
  remaining_amount NUMERIC(15,2) NOT NULL CHECK (remaining_amount >= 0),
  monthly_payment  NUMERIC(15,2) NOT NULL CHECK (monthly_payment > 0),
  due_date         DATE          NOT NULL,
  is_paid          BOOLEAN       NOT NULL DEFAULT FALSE,
  days_until_due   INT           NOT NULL DEFAULT 0,  -- stored: recalculated on every write
  created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_debts_family_id ON debts(family_id);
CREATE INDEX IF NOT EXISTS idx_debts_due_date  ON debts(due_date);

-- ── debt_payments ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS debt_payments (
  id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  debt_id    UUID          NOT NULL REFERENCES debts(id) ON DELETE CASCADE,
  paid_by    VARCHAR(50)   NOT NULL,
  amount     NUMERIC(15,2) NOT NULL CHECK (amount > 0),
  date       DATE          NOT NULL,
  created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_debt_payments_debt_id ON debt_payments(debt_id);

-- ── savings ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS savings (
  id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id        UUID          NOT NULL REFERENCES families(id) ON DELETE CASCADE,
  name             VARCHAR(150)  NOT NULL,
  target_amount    NUMERIC(15,2) NOT NULL CHECK (target_amount > 0),
  current_balance  NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (current_balance >= 0),
  target_date      DATE          NOT NULL,
  monthly_required NUMERIC(15,2) NOT NULL DEFAULT 0,  -- stored: recalculated on every write
  progress_pct     NUMERIC(6,2)  NOT NULL DEFAULT 0,  -- stored: current_balance / target_amount * 100
  created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_savings_family_id ON savings(family_id);

-- ── saving_deposits ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS saving_deposits (
  id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  saving_id  UUID          NOT NULL REFERENCES savings(id) ON DELETE CASCADE,
  amount     NUMERIC(15,2) NOT NULL CHECK (amount > 0),
  date       DATE          NOT NULL,
  note       TEXT          NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_saving_deposits_saving_id ON saving_deposits(saving_id);

-- ── recurring_budgets ─────────────────────────────────────────
-- Template anggaran berulang per kategori; otomatis di-apply tiap bulan
CREATE TABLE IF NOT EXISTS recurring_budgets (
  id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id     UUID          NOT NULL REFERENCES families(id) ON DELETE CASCADE,
  category_name VARCHAR(80)   NOT NULL,
  amount        NUMERIC(15,2) NOT NULL CHECK (amount >= 0),
  created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  UNIQUE (family_id, category_name)
);

CREATE INDEX IF NOT EXISTS idx_recurring_budgets_family_id ON recurring_budgets(family_id);

-- ── forward reference: transactions.debt_payment_id ──────────
-- debt_payments harus ada dulu sebelum transactions bisa di-FK,
-- tapi transactions juga di-FK dari debt_payments → lingkaran.
-- Solusi: tambahkan FK sebagai ALTER TABLE setelah keduanya ada.
ALTER TABLE transactions
  ADD CONSTRAINT fk_transactions_debt_payment
  FOREIGN KEY (debt_payment_id)
  REFERENCES debt_payments(id)
  ON DELETE SET NULL
  DEFERRABLE INITIALLY DEFERRED;

-- ── forward reference: transactions.saving_deposit_id ─────────
ALTER TABLE transactions
  ADD CONSTRAINT fk_transactions_saving_deposit
  FOREIGN KEY (saving_deposit_id)
  REFERENCES saving_deposits(id)
  ON DELETE SET NULL
  DEFERRABLE INITIALLY DEFERRED;
