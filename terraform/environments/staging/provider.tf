provider "aws" {
  region  = "ap-northeast-3"
  profile = "moku"

  default_tags {
    tags = {
      Environment = "staging"
      ManagedBy   = "terraform"
    }
  }
}
