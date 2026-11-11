terraform {
  required_version = ">= 1.6.0"

  required_providers {
    podman = {
      source  = "blechschmidt/podman"
      version = "~> 0.5"
    }
  }
}

provider "podman" {
  host = var.podman_socket
}
