data "aws_route53_zone" "this" {
  name = "mokudb-visualization.com"
}

resource "aws_route53_record" "api" {
  zone_id = data.aws_route53_zone.this.zone_id
  name    = "api.mokudb-visualization.com"
  type    = "A"
  ttl     = 300
  records = [aws_eip.this.public_ip]
}

resource "aws_route53_record" "root" {
  zone_id = data.aws_route53_zone.this.zone_id
  name    = "mokudb-visualization.com"
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.frontend.domain_name
    zone_id                = aws_cloudfront_distribution.frontend.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "acm_validation" {
  zone_id = data.aws_route53_zone.this.zone_id
  name    = "_3d9cc6223f4abbeb17920fa282c23786.mokudb-visualization.com"
  type    = "CNAME"
  ttl     = 300
  records = ["_8583bcd971caa463ced2699a0833c385.wzccmgtwzk.acm-validations.aws."]
}
