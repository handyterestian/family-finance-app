# Keuangan Keluarga — Family Finance App

Aplikasi manajemen keuangan keluarga berbasis web dengan arsitektur container penuh menggunakan protokol gRPC.

## Stack Teknologi

| Layer | Teknologi |
|---|---|
| Frontend | React 18 + Vite + Tailwind CSS |
| HTTP Gateway | Go + chi router |
| gRPC Server | Go + pgx/v5 |
| Database | PostgreSQL 15 |
| Infrastruktur | Docker Compose |

## Arsitektur

```
Browser (port 8080)
    ↓ HTTP/REST + Cookie
Nginx → React (SPA)
    ↓ /api/* proxy
Go HTTP Gateway (port 8000, internal)
    ↓ gRPC (port 50051, internal)
Go gRPC Server
    ↓ SQL
PostgreSQL (volume persisten)
```

## Prasyarat

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) v24+
- [Docker Compose](https://docs.docker.com/compose/) v2.20+

## Cara Menjalankan

```bash
# Clone / masuk ke folder
cd family-finance-app

# (Opsional) Salin dan sesuaikan environment variables
cp .env.example .env

# Build dan jalankan semua container
docker compose up --build

# Atau jalankan di background
docker compose up --build -d
```

Buka browser: **http://localhost:8080**

## Cara Menggunakan

1. **Daftar / Buat Keluarga** — Registrasi akun owner pertama dengan nama keluarga (seluruh anggota berbagi data domain keluarga).
2. **Login** — Masuk dengan email/username dan password.
3. **Data Contoh** — Klik tombol "📥 Data Contoh" di Dashboard untuk mengisi data demonstrasi secara instan.
4. **Navigasi** — Gunakan menu navigasi untuk mengakses Dashboard, Transaksi, Dompet, Anggaran, Dana Darurat, Hutang, Tabungan, Kategori, serta Admin Keluarga (khusus role owner).

## Reset / Hapus Semua Data

Klik tombol **"🗑 Hapus Semua"** di halaman Dashboard untuk mereset seluruh data transaksi dan perencanaan keluarga ke kondisi awal.

## Menghentikan Aplikasi

```bash
# Hentikan semua container
docker compose down

# Hentikan dan bersihkan volume database (data akan terhapus permanen)
docker compose down -v
```

## Fitur yang Diimplementasikan

### ✅ Multi-Dompet / Rekening & Transfer Antar Dompet
- Manajemen dompet / akun bank / e-wallet per keluarga.
- Transfer saldo antar dompet tercatat dalam riwayat transfer.
- Integrasi dompet pada setiap pencatatan transaksi masuk dan keluar.

### ✅ Transaksi (Pemasukan & Pengeluaran)
- Catat pemasukan dan pengeluaran (jumlah, dompet, kategori, pelaku/anggota, tanggal, catatan).
- Edit dan hapus transaksi.
- Pembayaran cicilan hutang dan setoran tabungan terintegrasi dengan saldo dompet.
- Filter riwayat transaksi berdasarkan rentang tanggal, kategori, atau anggota.

### ✅ Anggaran & Perencanaan Bulanan
- Penetapan anggaran bulanan per kategori (set, ubah, hapus).
- Pemantauan real-time progress pengeluaran vs batas anggaran.
- Anggaran berulang (*recurring budgets*) otomatis per bulan.

### ✅ Dana Darurat (Emergency Fund)
- Perhitungan target dana darurat ideal berdasarkan pengeluaran rata-rata bulanan.
- Setoran dan penarikan dana darurat dengan riwayat lengkap.

### ✅ Manajemen Hutang & Piutang
- Pencatatan hutang dengan informasi nominal, cicilan per bulan, dan tanggal jatuh tempo.
- Pembayaran cicilan hutang dengan pemotongan saldo dompet.
- Status progres pelunasan hutang dan badge peringatan jatuh tempo (< 7 hari).

### ✅ Target Tabungan (Savings Goals)
- Pembuatan target tabungan terarah berdasarkan tujuan dan target tanggal capaian.
- Setoran tabungan berkala dan kalkulasi rekomendasi nominal tabungan bulanan.

### ✅ Dashboard & Skor Kesehatan Keuangan
- Skor kesehatan finansial (0–100) berbasis 4 indikator kesehatan keuangan.
- Rekomendasi/saran otomatis berdasarkan aspek keuangan yang paling membutuhkan perhatian.
- Ringkasan pemasukan, pengeluaran, cash flow, serta visualisasi analitik per kategori & anggota.

### ✅ Manajemen Kategori Kustom
- Pembuatan, pembaruan, dan penghapusan kategori pengeluaran/pemasukan kustom.
- Penyediaan kategori bawaan (*default*) secara otomatis saat keluarga baru dibuat.

### ✅ Manajemen Anggota & Administrasi Keluarga (Role-based)
- Sistem peran akun: **Owner** (Kepala Keluarga / Admin) dan **Member** (Anggota).
- Admin panel untuk manajemen anggota keluarga, pergantian peran, serta pengaturan keluarga.
- Fitur auto backup & restore basis data.

## Struktur Proyek

```
family-finance-app/
├── proto/
│   └── finance.proto          # Kontrak gRPC (8 service)
├── server/                    # Go gRPC Server
│   ├── main.go
│   ├── db/db.go
│   ├── helpers/session.go
│   ├── services/              # auth, wallet, transaction, budget, debt, saving, category, dashboard, autobackup
│   ├── schema.sql             # DDL PostgreSQL (15 tabel)
│   └── Dockerfile
├── gateway/                   # Go HTTP Gateway
│   ├── main.go
│   ├── grpcclient/
│   ├── middleware/
│   ├── handlers/              # auth, wallets, transactions, budgets, debts, savings, categories, dashboard
│   └── Dockerfile
├── client/                    # React 18 + Vite + Tailwind CSS
│   ├── src/
│   │   ├── pages/             # Dashboard, Transactions, Wallet, Budgets, EmergencyFund, Debts, Savings, Categories, Admin
│   │   └── components/
│   ├── nginx.conf
│   └── Dockerfile
├── docker-compose.yml
└── .env.example
```
