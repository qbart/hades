output "app_server_names" {
  description = "Names of app servers."
  value       = [for s in hcloud_server.app : s.name]
}

output "db_server_names" {
  description = "Names of db servers."
  value       = [for s in hcloud_server.db : s.name]
}

output "app_server_ipv4" {
  description = "IPv4 addresses of app servers."
  value       = [for s in hcloud_server.app : s.ipv4_address]
}

output "db_server_ipv4" {
  description = "IPv4 addresses of db servers."
  value       = [for s in hcloud_server.db : s.ipv4_address]
}
