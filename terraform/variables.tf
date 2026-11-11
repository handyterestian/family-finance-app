variable "podman_socket" {
  description = "Podman Docker-compatible API socket as seen by the Terraform process."
  type        = string
  default     = "unix:///tmp/podman.sock"
}

variable "project_name" {
  description = "Prefix used for image, container and network names."
  type        = string
  default     = "family-finance"
}

variable "image_tag" {
  description = "Tag applied to locally built images."
  type        = string
  default     = "latest"
}

variable "http_port" {
  description = "Host port that exposes the React client through Nginx."
  type        = number
  default     = 8080
}

variable "db_user" {
  description = "PostgreSQL user."
  type        = string
  default     = "finance_user"
}

variable "db_password" {
  description = "PostgreSQL password."
  type        = string
  default     = "finance_pass"
  sensitive   = true
}

variable "db_name" {
  description = "PostgreSQL database name."
  type        = string
  default     = "finance_db"
}

variable "cors_origin" {
  description = "Origin allowed by the HTTP gateway."
  type        = string
  default     = "http://localhost:8080"
}
