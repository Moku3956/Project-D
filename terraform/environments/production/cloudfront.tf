resource "aws_cloudfront_origin_access_control" "frontend" {
  name                              = "oac-mokudb-frontend-prod.s3.ap-northeast-3.amazonaws-mtz9rprqcif"
  description                       = "Created by CloudFront"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "frontend" {
  comment             = "mokudb production"
  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"
  http_version        = "http2"
  price_class         = "PriceClass_200"
  aliases             = ["mokudb-visualization.com"]

  origin {
    domain_name              = aws_s3_bucket.frontend.bucket_regional_domain_name
    origin_id                = "mokudb-frontend-prod.s3.ap-northeast-3.amazonaws.com-mtz9rhk1v4i"
    origin_access_control_id = aws_cloudfront_origin_access_control.frontend.id
  }

  default_cache_behavior {
    target_origin_id       = "mokudb-frontend-prod.s3.ap-northeast-3.amazonaws.com-mtz9rhk1v4i"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["HEAD", "GET"]
    cached_methods          = ["HEAD", "GET"]
    compress               = true
    cache_policy_id         = "658327ea-f89d-4fab-a63d-7e88639e58f6"
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate.frontend.arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }

  tags = {
    Name = "mokudb-frontend-prod-cdn"
  }
}
