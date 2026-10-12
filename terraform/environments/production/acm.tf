provider "aws" {
  alias   = "us_east_1"
  region  = "us-east-1"
  profile = "moku"

  default_tags {
    tags = {
      Environment = "production"
      ManagedBy   = "terraform"
    }
  }
}

resource "aws_acm_certificate" "frontend" {
  provider          = aws.us_east_1
  domain_name       = "mokudb-visualization.com"
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}
