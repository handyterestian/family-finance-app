# Keuangan Keluarga — Family Finance App

Aplikasi manajemen keuangan keluarga berbasis web modern dengan arsitektur microservices container penuh menggunakan protokol gRPC performa tinggi.

---

## 🛠 Stack Teknologi

| Layer | Komponen & Teknologi | Deskripsi |
|---|---|---|
| **Frontend** | React 18 + Vite + Tailwind CSS | Single Page Application (SPA) responsif & modular |
| **Web Server / Reverse Proxy** | Nginx Alpine | Melayani build React SPA dan proxy endpoint `/api/*` |
| **API Gateway** | Go (chi router) | HTTP REST API Gateway, validasi cookie session, dan translasi gRPC client |
| **Core Service (gRPC)** | Go 1.23 + pgx/v5 | Server backend bisnis logika dengan komunikasi gRPC biner |
| **Protokol Komunikasi** | Protocol Buffers (proto3) | Kontrak antarmuka service data yang terstruktur & type-safe |
| **Database** | PostgreSQL 15 Alpine | Database relasional persisten dengan ekstensi `pgcrypto` |
| **Orkestrasi** | Docker Compose | Manajemen multi-container terisolasi dalam satu private network |

---

## 🏗 Arsitektur Sistem

```
Browser (Port 8080)
    │
    ▼ HTTP / REST + Cookie Session
┌────────────────────────────────────────────────────────┐
│ Nginx (Reverse Proxy & Static Web Server)              │
│ ├── /*        → React SPA (HTML/JS/CSS)               │
│ └── /api/*    → Proxy ke Go HTTP Gateway (Internal)    │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼ HTTP / JSON
┌────────────────────────────────────────────────────────┐
│ Go HTTP Gateway (Port 8000, Internal)                  │
│ ├── Middleware Auth & Session Management               │
│ ├── Request Validation & REST Routing (chi)            │
│ └── gRPC Client Interceptor                            │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼ gRPC Binary (Port 50051, Internal)
┌────────────────────────────────────────────────────────┐
│ Go gRPC Server (Port 50051, Internal)                  │
│ ├── 8 Domain Services (Auth, Wallet, Transaksi, dll)   │
│ ├── Auto-Backup Scheduler Mingguan                    │
│ └── Connection Pool Database (pgx/v5)                  │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼ SQL Queries / pgcrypto
┌────────────────────────────────────────────────────────┐
│ PostgreSQL 15 (Port 5432, Internal Volume Persisten)   │
│ └── 15 Relational Tables with Foreign Key Constraints  │
└────────────────────────────────────────────────────────┘
```

---

## 📋 Prasyarat Sistem

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) v24+ (Windows / macOS / Linux)
- [Docker Compose](https://docs.docker.com/compose/) v2.20+
- Port `8080` tersedia di mesin host

---

## 🚀 Panduan Menjalankan Aplikasi

### 1. Masuk ke Direktori Proyek
```bash
cd family-finance-app
```

### 2. (Opsional) Salin Environment Variables
```bash
cp .env.example .env
```

### 3. Build & Jalankan Container
```bash
# Build dan jalankan seluruh container
docker compose up --build

# Atau jalankan di background (detached mode)
docker compose up --build -d
```

### 4. Buka Aplikasi di Browser
Akses antarmuka web melalui browser:
👉 **[http://localhost:8080](http://localhost:8080)**

---

## 📖 Alur Penggunaan Aplikasi

1. **Pendaftaran (Register):** Buat akun pertama sebagai *Owner* sekaligus mendaftarkan Nama Keluarga. Seluruh anggota keluarga akan berbagi data keuangan yang sama.
2. **Login & Navigasi:** Masuk menggunakan email dan password yang telah didaftarkan.
3. **Data Demonstrasi (Seed Data):** Klik tombol **"📥 Data Contoh"** di halaman Dashboard untuk langsung mencoba aplikasi dengan contoh transaksi, anggaran, tabungan, dan dompet terisi.
4. **Pencatatan Keuangan Harian:** Catat mutasi kas masuk/keluar melalui menu Transaksi atau form cepat di Dashboard.
5. **Reset Data:** Klik tombol **"🗑 Hapus Semua"** di Dashboard jika ingin membersihkan seluruh data contoh dan memulai pencatatan dari nol.

---

## 🛑 Menghentikan & Membersihkan Container

```bash
# Menghentikan container tanpa menghapus data database
docker compose down

# Menghentikan container dan MENGHAPUS volume database (reset total)
docker compose down -v
```

---

## ✨ Fitur-Fitur Lengkap Aplikasi

### 1. 💰 Multi-Dompet / Rekening & Transfer Saldo
- **Manajemen Akun/Dompet:** Mendukung berbagai tipe akun: *Tunai (Cash), Bank, E-Wallet, Investasi, dan Lainnya*.
- **Kustomisasi Visual:** Pemilihan kode warna label per dompet untuk kemudahan identifikasi di UI.
- **Transfer Antar Dompet:** Fitur perpindahan saldo antar dompet (misal: penarikan ATM ke tunai, top-up e-wallet) disertai pencatatan histori transfer.
- **Sinkronisasi Saldo Otomatis:** Saldo dompet otomatis bertambah/berkurang saat transaksi dicatat, cicilan hutang dibayar, atau tabungan disetor.

### 2. 💳 Manajemen Transaksi (Pemasukan & Pengeluaran)
- **Multi-Domain Input:** Form pencatatan lengkap yang mendukung transaksi reguler, pembayaran cicilan hutang, dana darurat, dan setoran tabungan.
- **Atribusi Pelaku Transaksi:** Setiap pencatatan transaksi dapat diatribusikan ke anggota keluarga terkait (Ayah, Ibu, Anak, dll).
- **Filter & Rekapitulasi:** Filter data berdasarkan bulan (YYYY-MM), kategori, atau anggota keluarga.
- **Perhitungan Saldo Real-Time:** Menampilkan ringkasan total pemasukan, total pengeluaran, serta net cashflow per bulan.

### 3. 📋 Anggaran Bulanan & Anggaran Berulang (Recurring Budgets)
- **Batas Pengeluaran per Kategori:** Penetapan batas belanja per kategori pada bulan tertentu.
- **Indikator Visual Progress:** Progress bar dinamis (Aman / Waspada / Melebihi Batas) saat pengeluaran mendekati atau melampaui limit anggaran.
- **Master Anggaran Berulang (Recurring):** Simpan template anggaran bulanan tetap yang dapat diterapkan secara otomatis (*apply*) ke bulan berikutnya dengan satu klik.
- **Salin Anggaran Bulan Lalu:** Fitur duplikasi seluruh anggaran dari bulan sebelumnya.

### 4. 🛡️ Dana Darurat (Emergency Fund)
- **Kalkulator Target Cerdas:** Menghitung target nominal ideal dana darurat secara otomatis berdasarkan rata-rata pengeluaran bulanan keluarga dikalikan target bulan proteksi (default: 6 bulan).
- **Metrik Proteksi (Months Covered):** Menampilkan berapa bulan keluarga dapat bertahan hidup dengan saldo dana darurat saat ini.
- **Mutasi Setoran & Penarikan:** Riwayat transparan seluruh mutasi setor/tarik dana darurat beserta catatan keperluannya.

### 5. 🏦 Manajemen Hutang & Pembayaran Cicilan
- **Pencatatan Pinjaman:** Catat pinjaman/kewajiban dengan rincian total hutang, perkiraan cicilan bulanan, dan tanggal jatuh tempo.
- **Pelacakan Sisa Pokok:** Pelacakan otomatis sisa hutang dan persentase pelunasan.
- **Badge Peringatan Jatuh Tempo:** Peringatan visual otomatis apabila tanggal jatuh tempo kurang dari 7 hari.
- **Pembayaran Cicilan:** Catat pembayaran cicilan (siapa yang membayar & sumber dompet) yang otomatis membuat transaksi pengeluaran.

### 6. 🎯 Target Tabungan & Impian (Savings Goals)
- **Perencanaan Target:** Buat target impian (misal: Liburan, Renovasi, Pendidikan) dengan nominal target dan batas tanggal capaian.
- **Rekomendasi Setoran Bulanan:** Algoritma menghitung otomatis berapa nominal yang harus disisihkan per bulan agar target tercapai tepat waktu.
- **Setoran Berkala:** Form setoran tabungan dengan histori mutasi dan grafik progres capaian target.

### 7. 📊 Dashboard Finansial & Financial Health Score
- **Skor Kesehatan Keuangan (0–100):** Analisis otomatis & komprehensif berdasarkan 4 pilar finansial utama:
  - *Rasio Tabungan (Maks 30 poin):* Menilai progres akumulasi tabungan terhadap target yang ditentukan.
  - *Disiplin Anggaran (Maks 25 poin):* Mengukur kepatuhan pengeluaran terhadap batas pos anggaran per kategori tanpa overbudget.
  - *Kecukupan Dana Darurat (Maks 25 poin):* Mengukur kesiapan proteksi kas darurat (rasio bulan cadangan terhadap target ideal 6 bulan).
  - *Rasio Beban Hutang (Maks 20 poin):* Menganalisis *Debt Service Ratio* (porsi cicilan bulanan terhadap total pemasukan keluarga, ambang aman $\le$ 30%).
- **Visualisasi Progress Ring & Status Badge:** Indikator visual dinamis (*Sangat Baik, Sehat, Cukup, Perlu Perhatian*) beserta breakdown progress per pilar di antarmuka Dashboard.
- **Saran & Rekomendasi Pintar:** Algoritma deteksi kondisi kritis (beban cicilan >40%, overbudget >50%, proteksi <1 bulan) dan saran perbaikan spesifik berbasis pilar finansial terlemah.
- **Visualisasi Analitik:** Grafik pengeluaran per kategori belanja dan kontribusi pengeluaran per anggota keluarga.

### 8. 🏷️ Kategori Kustom
- Fleksibilitas menambah, mengedit nama, dan menghapus kategori per keluarga.
- Tipe kategori: *Pemasukan (Income), Pengeluaran (Expense), atau Keduanya (Both)*.
- Inisialisasi otomatis kategori standar saat pertama kali keluarga mendaftar.

### 9. ⚙️ Administrasi Keluarga, Role & Cadangan Data
- **Role-Based Access:** Mendukung peran **Owner** (Kepala Keluarga / Admin) dan **Member** (Anggota).
- **Cadangan Manual (Export JSON):** Unduh seluruh data keluarga ke dalam satu berkas format JSON terstruktur.
- **Pemulihan Data (Import / Restore):** Unggah berkas cadangan JSON untuk memulihkan seluruh data keluarga secara instan.
- **Auto-Backup Mingguan:** Server menjalankan scheduler otomatis yang mencadangkan basis data setiap 7 hari sekali dengan mekanisme overwrite aman untuk menghemat storage.

---

## 📁 Struktur Direktori Proyek

```
family-finance-app/
├── proto/
│   └── finance.proto                # Kontrak gRPC Interface (8 Services & Message Definitions)
│
├── server/                          # Core Go gRPC Backend Server
│   ├── main.go                      # Entrypoint gRPC Server & Registry
│   ├── db/
│   │   └── db.go                    # PostgreSQL Connection Pool (pgx/v5)
│   ├── helpers/
│   │   └── session.go               # Validasi & Autentikasi Sesi Database
│   ├── services/                    # Implementasi 8 gRPC Services
│   │   ├── auth.go                  # Layanan Registrasi, Login & Session
│   │   ├── wallet.go                # Layanan Dompet & Mutasi Transfer
│   │   ├── transaction.go           # Layanan Transaksi Masuk & Keluar
│   │   ├── budget.go                # Layanan Anggaran & Dana Darurat
│   │   ├── debt.go                  # Layanan Manajemen Hutang & Cicilan
│   │   ├── saving.go                # Layanan Target Tabungan
│   │   ├── category.go              # Layanan Kategori Kustom
│   │   ├── dashboard.go             # Kalkulasi Skor Kesehatan, Seed & Backup
│   │   └── autobackup.go            # Scheduler Cadangan Database Mingguan
│   ├── schema.sql                   # Skema DDL Database PostgreSQL (15 Tabel)
│   └── Dockerfile                   # Multi-stage Dockerfile (Go Compiler + Runtime)
│
├── gateway/                         # Go HTTP REST API Gateway
│   ├── main.go                      # HTTP Server & Route Definitions (chi router)
│   ├── grpcclient/
│   │   └── client.go                # Koneksi & Pemanggilan gRPC Client
│   ├── middleware/
│   │   └── auth.go                  # Middleware Validasi Cookie & Header Sesi
│   ├── handlers/                    # HTTP Handlers (Translasi JSON <-> gRPC)
│   │   ├── auth.go
│   │   ├── wallets.go
│   │   ├── transactions.go
│   │   ├── budgets.go
│   │   ├── debts.go
│   │   ├── savings.go
│   │   ├── categories.go
│   │   ├── dashboard.go
│   │   └── helpers.go
│   └── Dockerfile                   # Multi-stage Dockerfile Gateway
│
├── client/                          # React Frontend Application (SPA)
│   ├── src/
│   │   ├── pages/                   # Halaman Antarmuka Web
│   │   │   ├── DashboardPage.jsx    # Ringkasan, Skor Kesehatan & Quick Tx
│   │   │   ├── TransactionsPage.jsx # Mutasi & Riwayat Transaksi
│   │   │   ├── WalletPage.jsx       # Kelola Dompet & Transfer Saldo
│   │   │   ├── BudgetsPage.jsx      # Limit Anggaran & Recurring Budgets
│   │   │   ├── EmergencyFundPage.jsx# Target & Mutasi Dana Darurat
│   │   │   ├── DebtsPage.jsx        # Hutang & Pembayaran Cicilan
│   │   │   ├── SavingsPage.jsx      # Target Tabungan & Setoran
│   │   │   ├── CategoriesPage.jsx   # Kelola Kategori Kustom
│   │   │   ├── AdminPage.jsx        # Backup & Restore Data
│   │   │   ├── LoginPage.jsx        # Form Masuk Akun
│   │   │   └── RegisterPage.jsx     # Form Pendaftaran Keluarga Baru
│   │   ├── components/              # Komponen Reusable (Layout, Modal, dll)
│   │   ├── api.js                   # Axios HTTP Client Configuration
│   │   └── utils.js                 # Helper Format Rupiah, Tanggal, dll
│   ├── nginx.conf                   # Konfigurasi Nginx Web Server & Proxy
│   └── Dockerfile                   # Multi-stage Dockerfile (Node Build + Nginx)
│
├── docker-compose.yml               # Definisi Orkestrasi Multi-Container
└── .env.example                     # Template Konfigurasi Environment Variable
```
