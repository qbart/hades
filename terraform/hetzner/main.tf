provider "hcloud" {
  token = var.hcloud_token
}

resource "hcloud_server" "app" {
  count = var.app_count

  name        = format("%s-app-%02d", var.name_prefix, count.index + 1)
  server_type = var.server_type
  image       = var.image
  location    = var.location

  labels = merge(var.common_labels, {
    cluster = "app"
    env = "dev"
  })
}

resource "hcloud_server" "db" {
  count = var.db_count

  name        = format("%s-db-%02d", var.name_prefix, count.index + 1)
  server_type = var.server_type
  image       = var.image
  location    = var.location

  labels = merge(var.common_labels, {
    cluster = "db"
    env = "dev"
  })
}
