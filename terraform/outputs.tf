output "url" {
  description = "Open the React app at this address."
  value       = "http://localhost:${var.http_port}"
}

output "network" {
  value = podman_network.finance.name
}

output "containers" {
  value = {
    db      = podman_container.db.name
    server  = podman_container.server.name
    gateway = podman_container.gateway.name
    client  = podman_container.client.name
  }
}

output "images" {
  value = {
    db      = podman_image.db.name
    server  = podman_image.server.name
    gateway = podman_image.gateway.name
    client  = podman_image.client.name
  }
}

output "database_volume" {
  value = podman_volume.pgdata.name
}

output "backup_volume" {
  value = podman_volume.backups.name
}
