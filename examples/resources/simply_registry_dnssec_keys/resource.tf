resource "simply_registry_dnssec_keys" "example" {
  product = "example.com"

  keys = [
    {
      type = "ds"
      data = "12345 13 2 49fd46e6c4b45c55d4ac69c9c59cb3f3d2c3e91c10db85b2e3f1c12c4a0e3f1c"
    },
  ]
}
