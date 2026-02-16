variable "hcloud_token" {
  description = "Hetzner Cloud API token."
  type        = string
  sensitive   = true
}

variable "name_prefix" {
  description = "Prefix used for all server names."
  type        = string
  default     = "hades"
}

variable "app_count" {
  description = "Number of app servers."
  type        = number
  default     = 7
}

variable "db_count" {
  description = "Number of db servers."
  type        = number
  default     = 3
}

variable "server_type" {
  description = "Hetzner server type."
  type        = string
  default     = "cx22"
}

variable "image" {
  description = "Hetzner image."
  type        = string
  default     = "ubuntu-24.04"
}

variable "location" {
  description = "Hetzner location."
  type        = string
  default     = "fsn1"
}

variable "common_labels" {
  description = "Labels applied to both app and db servers."
  type        = map(string)
  default = {
    managed_by = "terraform"
  }
}
