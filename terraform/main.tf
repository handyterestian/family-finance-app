locals {
  repo_root = abspath("${path.module}/..")

  # Content hashes make `terraform apply` rebuild an image whenever its
  # source changes, without needing a separate make target.
  server_inputs = sha256(join("", concat(
    [for f in sort(fileset("${local.repo_root}/proto", "**")) : filesha256("${local.repo_root}/proto/${f}")],
    [for f in sort(fileset("${local.repo_root}/server", "**")) : filesha256("${local.repo_root}/server/${f}")],
  )))

  gateway_inputs = sha256(join("", concat(
    [for f in sort(fileset("${local.repo_root}/proto", "**")) : filesha256("${local.repo_root}/proto/${f}")],
    [for f in sort(fileset("${local.repo_root}/gateway", "**")) : filesha256("${local.repo_root}/gateway/${f}")],
  )))

  client_inputs = sha256(join("", concat(
    [for f in sort(fileset("${local.repo_root}/client/src", "**")) : filesha256("${local.repo_root}/client/src/${f}")],
    [for f in [
      "package.json",
      "index.html",
      "vite.config.js",
      "tailwind.config.js",
      "postcss.config.js",
      "nginx.conf",
      "Containerfile",
    ] : filesha256("${local.repo_root}/client/${f}")],
  )))

  db_inputs = sha256(join("", [
    filesha256("${local.repo_root}/db/Containerfile"),
    filesha256("${local.repo_root}/server/schema.sql"),
  ]))

  network_name = "${var.project_name}-net"
  db_image     = "localhost/${var.project_name}-db:${var.image_tag}"
  server_image = "localhost/${var.project_name}-server:${var.image_tag}"
  gateway_image = "localhost/${var.project_name}-gateway:${var.image_tag}"
  client_image = "localhost/${var.project_name}-client:${var.image_tag}"
}

# ── Podman network ────────────────────────────────────────────
resource "podman_network" "finance" {
  name = local.network_name
}

# ── Persistent volumes ────────────────────────────────────────
resource "podman_volume" "pgdata" {
  name = "${var.project_name}-pgdata"
}

resource "podman_volume" "backups" {
  name = "${var.project_name}-backups"
}

# ── Images (built from Containerfiles) ────────────────────────
resource "podman_image" "db" {
  name         = local.db_image
  keep_locally = true

  build {
    context    = local.repo_root
    dockerfile = "db/Containerfile"
    tag        = [local.db_image]
  }

  triggers = {
    source = local.db_inputs
  }
}

resource "podman_image" "server" {
  name         = local.server_image
  keep_locally = true

  build {
    context    = local.repo_root
    dockerfile = "server/Containerfile"
    tag        = [local.server_image]
  }

  triggers = {
    source = local.server_inputs
  }
}

resource "podman_image" "gateway" {
  name         = local.gateway_image
  keep_locally = true

  build {
    context    = local.repo_root
    dockerfile = "gateway/Containerfile"
    tag        = [local.gateway_image]
  }

  triggers = {
    source = local.gateway_inputs
  }
}

resource "podman_image" "client" {
  name         = local.client_image
  keep_locally = true

  build {
    context    = local.repo_root
    dockerfile = "client/Containerfile"
    tag        = [local.client_image]
  }

  triggers = {
    source = local.client_inputs
  }
}

# ── PostgreSQL ────────────────────────────────────────────────
resource "podman_container" "db" {
  name    = "finance_db"
  image   = podman_image.db.image_id
  restart = "unless-stopped"

  env = [
    "POSTGRES_USER=${var.db_user}",
    "POSTGRES_PASSWORD=${var.db_password}",
    "POSTGRES_DB=${var.db_name}",
  ]

  volumes {
    volume_name    = podman_volume.pgdata.name
    container_path = "/var/lib/postgresql"
  }

  networks_advanced {
    name    = podman_network.finance.name
    aliases = ["postgres"]
  }

  healthcheck {
    test     = ["CMD-SHELL", "pg_isready -U ${var.db_user} -d ${var.db_name}"]
    interval = "5s"
    timeout  = "5s"
    retries  = 12
  }
}

# ── Go gRPC server ────────────────────────────────────────────
resource "podman_container" "server" {
  name     = "finance_server"
  image    = podman_image.server.image_id
  restart  = "unless-stopped"
  must_run = false

  env = [
    "DB_HOST=postgres",
    "DB_PORT=5432",
    "DB_USER=${var.db_user}",
    "DB_PASSWORD=${var.db_password}",
    "DB_NAME=${var.db_name}",
    "GRPC_PORT=50051",
    "BACKUP_DIR=/app/backups",
  ]

  volumes {
    volume_name    = podman_volume.backups.name
    container_path = "/app/backups"
  }

  networks_advanced {
    name    = podman_network.finance.name
    aliases = ["server"]
  }

  depends_on = [podman_container.db]
}

# ── Go HTTP gateway ───────────────────────────────────────────
resource "podman_container" "gateway" {
  name     = "finance_gateway"
  image    = podman_image.gateway.image_id
  restart  = "unless-stopped"
  must_run = false

  env = [
    "GRPC_HOST=server",
    "GRPC_PORT=50051",
    "GATEWAY_PORT=8000",
    "CORS_ORIGIN=${var.cors_origin}",
  ]

  networks_advanced {
    name    = podman_network.finance.name
    aliases = ["gateway"]
  }

  depends_on = [podman_container.server]
}

# ── React client + Nginx ──────────────────────────────────────
resource "podman_container" "client" {
  name     = "finance_client"
  image    = podman_image.client.image_id
  restart  = "unless-stopped"
  must_run = false

  ports {
    internal = 8080
    external = var.http_port
    protocol = "tcp"
  }

  networks_advanced {
    name    = podman_network.finance.name
    aliases = ["client"]
  }

  depends_on = [podman_container.gateway]
}
