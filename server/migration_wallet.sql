-- ─────────────────────────────────────────────────────────────
-- Migration: tambah tabel wallets, wallet_transfers,
--            dan kolom wallet_id di transactions
-- Jalankan di container DB yang sudah berjalan:
--   docker exec finance_db psql -U finance_user -d finance_db -f /migration_wallet.sql
-- Atau di dev:
--   docker exec finance_db_dev psql -U finance_user -d finance_db -f /migration_wallet.sql
-- ─────────────────────────────────────────────────────────────

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Buat tabel wallets (idempotent)
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

-- Buat tabel wallet_transfers (idempotent)
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

-- Tambah kolom wallet_id ke transactions (idempotent)
ALTER TABLE transactions
  ADD COLUMN IF NOT EXISTS wallet_id UUID NULL REFERENCES wallets(id) ON DELETE SET NULL;

-- Tambah kolom saving_deposit_id ke transactions (idempotent)
ALTER TABLE transactions
  ADD COLUMN IF NOT EXISTS saving_deposit_id UUID NULL;

-- Tambah FK saving_deposit_id hanya jika belum ada
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'fk_transactions_saving_deposit'
  ) THEN
    ALTER TABLE transactions
      ADD CONSTRAINT fk_transactions_saving_deposit
      FOREIGN KEY (saving_deposit_id)
      REFERENCES saving_deposits(id)
      ON DELETE SET NULL
      DEFERRABLE INITIALLY DEFERRED;
  END IF;
END$$;
