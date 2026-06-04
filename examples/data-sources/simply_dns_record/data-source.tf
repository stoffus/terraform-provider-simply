data "simply_dns_record" "www" {
  product = "example.com"
  name    = "www"
  type    = "CNAME"
}
