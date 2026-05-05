data "git_repository" "current" {}

data "git_repository" "explicit" {
  path = path.module
}

output "current_origin_url" {
  value = data.git_repository.current.origin_url
}

output "current_default_remote_branch" {
  value = data.git_repository.current.default_remote_branch
}
