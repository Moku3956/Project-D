terraform {
  required_version = ">= 1.16.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket       = "mokudb-terraform-state"
    key          = "production/terraform.tfstate"
    region       = "ap-northeast-3"
    profile      = "moku"
    use_lockfile = true
  }
}
