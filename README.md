# Keuangan Keluarga — Family Finance App

Aplikasi manajemen keuangan keluarga berbasis web modern dengan arsitektur microservices container penuh menggunakan protokol gRPC performa tinggi.

---

## 🛠 Stack Teknologi

| Layer | Komponen & Teknologi | Deskripsi |
|---|---|---|
| **Frontend** | React 18 + Vite 5 + Tailwind CSS 3 | Single Page Application (SPA) responsif & modular |
| **Web Server / Reverse Proxy** | Nginx Unprivileged 1.29 Alpine | Melayani build React SPA dan proxy endpoint `/api/*` |
| **API Gateway** | Go 1.23 (chi v5) | HTTP REST API Gateway, validasi cookie session, dan translasi gRPC client |
| **Core Service (gRPC)** | Go 1.23 + pgx/v5 | Server backend bisnis logika dengan komunikasi gRPC biner |
| **Protokol Komunikasi** | Protocol Buffers (proto3) | Kontrak antarmuka service data yang terstruktur & type-safe |
| **Database** | PostgreSQL 18 Alpine | Database relasional persisten dengan ekstensi `pgcrypto` |
| **Orkestrasi** | Podman + Terraform | Build image & manajemen container/network/volume secara deklaratif (IaC) |
| **Image Build** | Go 1.25, Node 24, Alpine 3.22 | Multi-stage build di dalam Containerfile masing-masing service |

---

## 🏗 Arsitektur Sistem

```
Browser (Port 8080)
    │
    ▼ HTTP / REST + Cookie Session
┌────────────────────────────────────────────────────────┐
│ Nginx Unprivileged (Reverse Proxy & Static Web Server) │
│ ├── /*        → React SPA (HTML/JS/CSS)               │
│ └── /api/*    → Proxy ke Go HTTP Gateway (Internal)    │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼ HTTP / JSON
┌────────────────────────────────────────────────────────┐
│ Go HTTP Gateway (Port 8000, Internal)                  │
│ ├── Middleware Auth & Session Cookie                   │
│ ├── Request Validation & REST Routing (chi)            │
│ ├── CORS Handler                                       │
│ └── gRPC Client Stubs                                  │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼ gRPC Binary (Port 50051, Internal)
┌────────────────────────────────────────────────────────┐
│ Go gRPC Server (Port 50051, Internal)                  │
│ ├── 8 Domain Services (Auth, Wallet, Transaksi, dll)   │
│ ├── Auto-Backup Scheduler Mingguan                    │
│ ├── gRPC Reflection (untuk grpcurl)                    │
│ └── Connection Pool Database (pgx/v5)                  │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼ SQL Queries / pgcrypto
┌────────────────────────────────────────────────────────┐
│ PostgreSQL 18 (Port 5432, Internal Volume Persisten)   │
│ └── 15 Relational Tables with Foreign Key Constraints  │
└────────────────────────────────────────────────────────┘
```

Semua service berjalan di network Podman privat `${project_name}-net`. Hanya container client (Nginx) yang mengekspos port ke host.

---

## 📋 Prasyarat Sistem

- [Podman](https://podman.io/) v4.4+ (rootless didukung)
- Systemd user session aktif (untuk `podman.socket`)
- Port `8080` tersedia di mesin host (dapat diubah lewat variabel `http_port`)

Terraform **tidak perlu di-install di host** — skrip `terraform.sh` menjalankannya di dalam container Podman sementara (`hashicorp/terraform:1.9`). Node.js dan Go juga tidak perlu di-install untuk menjalankan stack (semua build terjadi di dalam container).

---

## 🚀 Panduan Menjalankan Aplikasi

### 1. Masuk ke Direktori Proyek
```bash
cd family-finance-app
```

### 2. Aktifkan Podman API Socket
```bash
systemctl --user enable --now podman.socket
```

### 3. (Opsional) Sesuaikan Konfigurasi
```bash
cp terraform/terraform.tfvars.example terraform/terraform.tfvars
```

### 4. Build & Jalankan Seluruh Stack dengan Terraform
```bash
# Unduh provider Podman (blechschmidt/podman)
./terraform.sh init

# Build image dari Containerfile lalu buat network, volume, dan container
./terraform.sh apply -auto-approve
```

Perintah di atas membangun image `db`, `server`, `gateway`, dan `client`, lalu menjalankannya di network Podman privat. Image akan otomatis di-rebuild saat source terkait berubah (dideteksi lewat content hash di `terraform/main.tf`).

### 5. Buka Aplikasi di Browser
Akses antarmuka web melalui browser:
👉 **[http://localhost:8080](http://localhost:8080)**

### Perintah Berguna
```bash
./terraform.sh plan                 # Lihat perubahan yang akan diterapkan
./terraform.sh output               # Lihat URL, nama container, image, dan volume
./terraform.sh apply -auto-approve  # Rebuild image yang source-nya berubah + terapkan
./terraform.sh destroy              # Hentikan & hapus container/network (volume tetap)
podman ps                           # Cek status container
podman logs -f finance_server       # Lihat log server gRPC
podman logs -f finance_gateway      # Lihat log HTTP gateway
podman logs -f finance_client       # Lihat log Nginx
```

---

## 🔧 Konfigurasi

### Variabel Terraform (`terraform/terraform.tfvars`)

| Variabel | Default | Deskripsi |
|---|---|---|
| `project_name` | `family-finance` | Prefix nama image, container, dan network |
| `image_tag` | `latest` | Tag image hasil build |
| `http_port` | `8080` | Port host untuk mengakses client melalui Nginx |
| `db_user` | `finance_user` | User PostgreSQL |
| `db_password` | `finance_pass` | Password PostgreSQL (sensitive) |
| `db_name` | `finance_db` | Nama database PostgreSQL |
| `cors_origin` | `http://localhost:8080` | Origin yang diizinkan oleh gateway |
| `podman_socket` | `unix:///tmp/podman.sock` | Socket Podman yang dipakai Terraform |

### Environment Container (di-set otomatis oleh Terraform)

| Service | Environment |
|---|---|
| `finance_db` | `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` |
| `finance_server` | `DB_HOST=postgres`, `DB_PORT=5432`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `GRPC_PORT=50051`, `BACKUP_DIR=/app/backups` |
| `finance_gateway` | `GRPC_HOST=server`, `GRPC_PORT=50051`, `GATEWAY_PORT=8000`, `CORS_ORIGIN` |
| `finance_client` | Port internal `8080` dipetakan ke `http_port` host |

---

## 📖 Alur Penggunaan Aplikasi

1. **Pendaftaran (Register):** Buat akun sekaligus mendaftarkan Nama Keluarga. Setiap registrasi membuat keluarga baru dengan akun tersebut sebagai *Owner*, dan seluruh data domain terikat ke `family_id` keluarga tersebut.
2. **Login & Navigasi:** Masuk menggunakan email dan password yang telah didaftarkan. Sidebar menyediakan akses ke seluruh modul (Dashboard, Transaksi, Dompet, Anggaran, Dana Darurat, Hutang, Tabungan, Kategori, Admin).
3. **Siapkan Dompet:** Tambahkan dompet/rekening (Tunai, Bank, E-Wallet, Investasi, Lainnya) beserta saldo awal dan warna label.
4. **Pencatatan Keuangan Harian:** Catat mutasi kas masuk/keluar melalui menu **Transaksi** atau form cepat di **Dashboard**.
5. **Atur Anggaran & Target:** Tetapkan anggaran per kategori, buat template anggaran berulang, isi dana darurat, kelola hutang/cicilan, dan buat target tabungan.
6. **Pantau Kesehatan Finansial:** Dashboard menampilkan skor kesehatan keuangan (0–100), progress ring per pilar, grafik pengeluaran, serta saran perbaikan.
7. **Cadangkan Data:** Gunakan menu **Admin** untuk mengunduh backup JSON atau memulihkannya. Auto-backup mingguan juga berjalan otomatis di server.

> Catatan: endpoint demo **`POST /api/dashboard/seed`** dan **`DELETE /api/dashboard/clear`** masih tersedia di API (mis. untuk pengujian), namun tombolnya tidak lagi ditampilkan di antarmuka Dashboard.

---

## ✨ Fitur-Fitur Lengkap Aplikasi

### 1. 💰 Multi-Dompet / Rekening & Transfer Saldo
- **Manajemen Akun/Dompet:** Mendukung berbagai tipe akun: *Tunai (Cash), Bank, E-Wallet, Investasi, dan Lainnya*.
- **Kustomisasi Visual:** Pemilihan kode warna label per dompet untuk kemudahan identifikasi di UI.
- **Transfer Antar Dompet:** Fitur perpindahan saldo antar dompet (misal: penarikan ATM ke tunai, top-up e-wallet) disertai pencatatan histori transfer.
- **Penyesuaian Saldo (Adjust):** Koreksi saldo dompet secara manual; selisih otomatis dicatat sebagai transaksi "Penyesuaian Saldo".
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
- **Role-Based Access:** Skema database mendukung peran **Owner** (Kepala Keluarga / Admin) dan **Member** (Anggota); registrasi membuat pemilik akun sebagai Owner. Alur undang/gabung anggota belum tersedia di UI.
- **Cadangan Manual (Export JSON):** Unduh seluruh data keluarga ke dalam satu berkas format JSON terstruktur.
- **Pemulihan Data (Import / Restore):** Unggah berkas cadangan JSON untuk memulihkan seluruh data keluarga secara instan.
- **Auto-Backup Mingguan:** Server menjalankan scheduler otomatis yang mencadangkan basis data setiap 7 hari sekali dengan mekanisme overwrite aman untuk menghemat storage. File disimpan di volume `${project_name}-backups`.

### 10. 📱 Antarmuka Responsif
- Layout sidebar yang dapat dilipat pada layar kecil, top bar khusus mobile, dan tabel/daftar yang menyesuaikan ukuran perangkat.

---

## 🔌 Endpoint HTTP API

Semua endpoint di bawah ini diakses melalui Nginx pada prefix `/api/*` (prefix di-strip sebelum diteruskan ke gateway). Endpoint bertanda 🔒 membutuhkan cookie `session_id` yang valid.

| Method | Endpoint | Deskripsi |
|---|---|---|
| `GET` | `/api/health` | Health check gateway |
| `POST` | `/api/auth/register` | Registrasi keluarga & owner |
| `POST` | `/api/auth/login` | Login (set cookie `session_id`) |
| `POST` | `/api/auth/logout` 🔒 | Logout & hapus cookie |
| `GET` | `/api/auth/me` 🔒 | Info user yang sedang login |
| `GET`/`POST` | `/api/categories` 🔒 | List / buat kategori |
| `PUT`/`DELETE` | `/api/categories/{name}` 🔒 | Ubah / hapus kategori |
| `GET`/`POST` | `/api/wallets` 🔒 | List / buat dompet |
| `PUT`/`DELETE` | `/api/wallets/{id}` 🔒 | Ubah / hapus dompet |
| `POST` | `/api/wallets/transfer` 🔒 | Transfer antar dompet |
| `POST` | `/api/wallets/{id}/adjust` 🔒 | Penyesuaian saldo dompet |
| `GET`/`POST` | `/api/transactions` 🔒 | List / buat transaksi |
| `PUT`/`DELETE` | `/api/transactions/{id}` 🔒 | Ubah / hapus transaksi |
| `GET`/`POST` | `/api/budgets` 🔒 | List / set anggaran |
| `PUT`/`DELETE` | `/api/budgets/{id}` 🔒 | Ubah / hapus anggaran |
| `GET`/`POST` | `/api/budgets/recurring` 🔒 | List / set anggaran berulang |
| `POST` | `/api/budgets/recurring/apply` 🔒 | Terapkan template anggaran berulang |
| `DELETE` | `/api/budgets/recurring/{id}` 🔒 | Hapus template anggaran berulang |
| `GET`/`PUT` | `/api/budgets/emergency` 🔒 | Lihat / set dana darurat |
| `POST` | `/api/budgets/emergency/deposit` 🔒 | Setor / tarik dana darurat |
| `GET` | `/api/budgets/emergency/history` 🔒 | Riwayat mutasi dana darurat |
| `GET`/`POST` | `/api/debts` 🔒 | List / buat hutang |
| `PUT`/`DELETE` | `/api/debts/{id}` 🔒 | Ubah / hapus hutang |
| `POST` | `/api/debts/{id}/pay` 🔒 | Bayar cicilan hutang |
| `GET` | `/api/debts/{id}/payments` 🔒 | Riwayat pembayaran hutang |
| `GET`/`POST` | `/api/savings` 🔒 | List / buat target tabungan |
| `PUT`/`DELETE` | `/api/savings/{id}` 🔒 | Ubah / hapus target tabungan |
| `POST` | `/api/savings/{id}/deposit` 🔒 | Setor tabungan |
| `GET` | `/api/dashboard` 🔒 | Ringkasan dashboard & health score |
| `POST` | `/api/dashboard/seed` 🔒 | Isi data contoh |
| `DELETE` | `/api/dashboard/clear` 🔒 | Hapus seluruh data keluarga |
| `GET` | `/api/backup` 🔒 | Unduh backup JSON |
| `POST` | `/api/restore` 🔒 | Restore dari file backup JSON |

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
│   │   ├── wallet.go                # Layanan Dompet, Transfer & Penyesuaian Saldo
│   │   ├── transaction.go           # Layanan Transaksi Masuk & Keluar
│   │   ├── budget.go                # Layanan Anggaran, Anggaran Berulang & Dana Darurat
│   │   ├── debt.go                  # Layanan Manajemen Hutang & Cicilan
│   │   ├── saving.go                # Layanan Target Tabungan
│   │   ├── category.go              # Layanan Kategori Kustom
│   │   ├── dashboard.go             # Kalkulasi Skor Kesehatan, Seed & Backup
│   │   └── autobackup.go            # Scheduler Cadangan Database Mingguan
│   ├── generated/                   # Stub protobuf hasil generate (di-.gitignore)
│   ├── schema.sql                   # Skema DDL Database PostgreSQL (15 Tabel)
│   ├── migration_wallet.sql         # Migrasi untuk DB lama (tabel dompet & transfer)
│   └── Containerfile                # Multi-stage build (Go Compiler + Runtime)
│
├── gateway/                         # Go HTTP REST API Gateway
│   ├── main.go                      # HTTP Server, Middleware & Route Definitions (chi)
│   ├── grpcclient/
│   │   └── client.go                # Koneksi & Pemanggilan gRPC Client
│   ├── middleware/
│   │   └── auth.go                  # Middleware Validasi Cookie Session
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
│   ├── generated/                   # Stub protobuf hasil generate (di-.gitignore)
│   └── Containerfile                # Multi-stage build Gateway
│
├── client/                          # React Frontend Application (SPA)
│   ├── src/
│   │   ├── pages/                   # Halaman Antarmuka Web
│   │   │   ├── DashboardPage.jsx    # Ringkasan, Skor Kesehatan & Quick Tx
│   │   │   ├── TransactionsPage.jsx # Mutasi & Riwayat Transaksi
│   │   │   ├── WalletPage.jsx       # Kelola Dompet, Transfer & Adjust Saldo
│   │   │   ├── BudgetsPage.jsx      # Limit Anggaran & Recurring Budgets
│   │   │   ├── EmergencyFundPage.jsx# Target & Mutasi Dana Darurat
│   │   │   ├── DebtsPage.jsx        # Hutang & Pembayaran Cicilan
│   │   │   ├── SavingsPage.jsx      # Target Tabungan & Setoran
│   │   │   ├── CategoriesPage.jsx   # Kelola Kategori Kustom
│   │   │   ├── AdminPage.jsx        # Backup & Restore Data
│   │   │   ├── LoginPage.jsx        # Form Masuk Akun
│   │   │   └── RegisterPage.jsx     # Form Pendaftaran Keluarga Baru
│   │   ├── components/              # Komponen Reusable
│   │   │   ├── Layout.jsx           # Sidebar, Top Bar & Navigasi Responsif
│   │   │   └── Modal.jsx            # Dialog Modal Reusable
│   │   ├── hooks/
│   │   │   └── usePageFocus.js      # Refresh data saat halaman kembali difokuskan
│   │   ├── App.jsx                  # Router, Protected Route & UserContext
│   │   ├── api.js                   # Axios Client & Interceptor 401
│   │   ├── utils.js                 # Helper Format Rupiah, Tanggal, dll
│   │   ├── index.css                # Tailwind & Style Global
│   │   └── main.jsx                 # Entrypoint React
│   ├── nginx.conf                   # Konfigurasi Nginx Web Server & Proxy
│   ├── vite.config.js               # Konfigurasi Vite & Proxy Dev
│   ├── tailwind.config.js
│   ├── postcss.config.js
│   └── Containerfile                # Multi-stage build (Node Build + Nginx unprivileged)
│
├── db/
│   └── Containerfile                # Image PostgreSQL 18 + schema.sql di initdb.d
│
├── terraform/                       # Infrastruktur Podman sebagai kode (HCL)
│   ├── versions.tf                  # Terraform >= 1.6 & provider blechschmidt/podman ~> 0.5
│   ├── variables.tf                 # Port, kredensial DB, CORS, nama project
│   ├── main.tf                      # Network, volume, image build, container
│   ├── outputs.tf                   # URL, container, image, volume
│   └── terraform.tfvars.example     # Template konfigurasi
│
├── terraform.sh                     # Wrapper: jalankan Terraform di container Podman
├── .containerignore                 # File yang diabaikan saat build container
└── .dockerignore
```

---

## 💻 Pengembangan Lokal (Tanpa Terraform)

Prasyarat tambahan: Go 1.23+, Node.js 20+, dan `protoc` beserta plugin `protoc-gen-go` / `protoc-gen-go-grpc`.

### 1. Jalankan PostgreSQL
```bash
podman volume create finance-dev-pgdata
podman run -d --name finance-db-dev \
  -e POSTGRES_USER=finance_user \
  -e POSTGRES_PASSWORD=finance_pass \
  -e POSTGRES_DB=finance_db \
  -v finance-dev-pgdata:/var/lib/postgresql \
  -v "$PWD/server/schema.sql:/docker-entrypoint-initdb.d/01_schema.sql:ro" \
  -p 5432:5432 postgres:18-alpine
```

### 2. Generate Stub Protobuf
Direktori `generated/` di-`.gitignore` dan dibuat ulang saat build container. Untuk dev lokal:
```bash
mkdir -p server/generated gateway/generated

protoc --proto_path=proto \
  --go_out=server/generated --go_opt=paths=source_relative \
  --go-grpc_out=server/generated --go-grpc_opt=paths=source_relative \
  proto/finance.proto

protoc --proto_path=proto \
  --go_out=gateway/generated --go_opt=paths=source_relative \
  --go-grpc_out=gateway/generated --go-grpc_opt=paths=source_relative \
  proto/finance.proto
```

### 3. Jalankan Server, Gateway, dan Client
```bash
# Terminal 1 — gRPC server
cd server && go mod tidy && go run .

# Terminal 2 — HTTP gateway
cd gateway && go mod tidy && go run .

# Terminal 3 — React dev server
cd client && npm install && npm run dev
```

Vite berjalan di `http://localhost:5173` dan mem-proxy `/api` ke `http://localhost:8000` (dapat diubah lewat `VITE_GATEWAY_URL`).

> Jika database sudah ada sebelum fitur multi-dompet ditambahkan, jalankan `server/migration_wallet.sql` di container DB:
> ```bash
> podman exec -i finance_db psql -U finance_user -d finance_db < server/migration_wallet.sql
> ```

---

## 🛑 Menghentikan & Membersihkan Container

```bash
# Hentikan dan hapus container + network (data database di volume tetap aman)
./terraform.sh destroy -auto-approve

# Hapus total termasuk volume database & backup (reset penuh)
podman volume rm family-finance-pgdata family-finance-backups

# Hapus image hasil build
podman rmi localhost/family-finance-{db,server,gateway,client}:latest
```

---

## 📝 Catatan Teknis

- **Port & Network:** Hanya `client` yang dipublikasikan ke host (`http_port`, default 8080). `gateway`, `server`, dan `db` hanya dapat diakses antar-container melalui network Podman.
- **Session:** Autentikasi memakai cookie `session_id` (`HttpOnly`, `SameSite=Lax`, berlaku 7 hari) yang divalidasi gateway ke gRPC server pada setiap request terproteksi.
- **Data Isolation:** Semua tabel domain terikat ke `family_id`, sehingga seluruh anggota satu keluarga berbagi dataset yang sama.
- **Build Cache:** Terraform menghitung hash source dan hanya me-rebuild image yang berubah saat `apply`.
- **Generated Code:** `server/generated` dan `gateway/generated` tidak di-commit; Containerfile menjalankan `protoc` sebelum kompilasi.
