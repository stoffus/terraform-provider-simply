resource "simply_dns_record" "www" {
  product = "example.com"
  name    = "www"
  type    = "CNAME"
  data    = "web.example.net"
  ttl     = 3600
  comment = "Example web alias"
}
