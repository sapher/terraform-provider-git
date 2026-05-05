# Terraform Provider Git

The Git provider exposes information about local Git repositories.

## Example Usage

```hcl
terraform {
  required_providers {
    git = {
      source = "sapher/git"
    }
  }
}

data "git_repository" "current" {}

output "namespace" {
  value = data.git_repository.current.namespace
}

output "origin_url" {
  value = data.git_repository.current.origin_url
}

output "default_remote_branch" {
  value = data.git_repository.current.default_remote_branch
}
```

## Local Development

Run the Go test suite from the provider directory:

```sh
go test ./...
```

To test the provider with Terraform before publishing it, build a local provider binary and use Terraform CLI development overrides:

```sh
mkdir -p ./bin
go build -o ./bin/terraform-provider-git
```

Add a development override to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/sapher/git" = "/path/to/terraform-provider-git/bin"
  }

  direct {}
}
```

Replace `/path/to/terraform-provider-git/bin` with the absolute path to the `bin` directory you created locally.

Then run Terraform from a test configuration that uses `source = "sapher/git"`:

```sh
terraform plan
```

With `dev_overrides`, Terraform can use the local binary directly. You generally do not need `terraform init` for local provider development until the provider is available from a registry.

## Releasing

GitHub Actions publishes Terraform Registry-compatible release assets when a version tag is pushed:

```sh
git tag v0.1.0
git push origin v0.1.0
```

Configure these repository secrets before publishing the first release:

- `GPG_PRIVATE_KEY` Armored private key used to sign the checksum file.
- `PASSPHRASE` Passphrase for the private key, if one is set.

The release workflow builds provider archives with GoReleaser, uploads them to the GitHub release, and signs the checksum file required by the Terraform Registry.

## Data Sources

### `git_repository`

Reads remote-origin metadata from a local Git repository.

#### Arguments

This data source has no arguments. It reads the Git repository containing Terraform's current working module directory.

#### Attributes

- `namespace` Repository namespace and name extracted from `origin_url`, without a trailing `.git` suffix.
- `origin_url` First configured URL for the `origin` remote.
- `default_remote_branch` Best-effort default branch for `origin`, resolved from local Git metadata when available. The provider first reads `refs/remotes/origin/HEAD`, then falls back to the current branch's `origin` upstream. The value is null when no local default branch metadata is available.
