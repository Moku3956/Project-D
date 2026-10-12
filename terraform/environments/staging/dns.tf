data "aws_route53_zone" "this" {
  name = "mokudb-visualization.com"
}

resource "aws_route53_record" "api_stg" {
  zone_id = data.aws_route53_zone.this.zone_id
  name    = "api-stg.mokudb-visualization.com"
  type    = "A"
  ttl     = 300
  records = [aws_eip.this.public_ip]
}
