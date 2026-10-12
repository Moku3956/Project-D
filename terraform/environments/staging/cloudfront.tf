resource "aws_cloudfront_origin_access_control" "frontend" {
  name                              = "oac-mokudb-frontend-stg.s3.ap-northeast-3.amazonaws.-mtz8iythdzz"
  description                       = "Created by CloudFront"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "frontend" {
  comment             = "mokudb staging"
  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = ""
  http_version        = "http2"
  price_class         = "PriceClass_200"

  origin {
    domain_name              = aws_s3_bucket.frontend.bucket_regional_domain_name
    origin_id                = "mokudb-frontend-stg.s3.ap-northeast-3.amazonaws.com-mtz89lud6ts"
    origin_access_control_id = aws_cloudfront_origin_access_control.frontend.id
  }

  default_cache_behavior {
    target_origin_id       = "mokudb-frontend-stg.s3.ap-northeast-3.amazonaws.com-mtz89lud6ts"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["HEAD", "GET"]
    cached_methods          = ["HEAD", "GET"]
    compress               = true
    cache_policy_id         = "658327ea-f89d-4fab-a63d-7e88639e58f6"
  }

  custom_error_response {
    error_code         = 403
    response_code      = 200
    response_page_path = "/index.html"
    error_caching_min_ttl = 10
  }

  custom_error_response {
    error_code         = 404
    response_code      = 200
    response_page_path = "/index.html"
    error_caching_min_ttl = 10
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  tags = {
    Name = "mokudb-frontend-stg-cdn"
  }
}
