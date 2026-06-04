resource "simply_registry_nameservers" "example" {
  product = "example.com"

  nameservers = [
    "ns1.simply.com",
    "ns2.simply.com",
  ]
}
