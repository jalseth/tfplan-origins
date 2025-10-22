resource "local_file" "baz" {
    filename = "foobarbaz"
    content  = "inside a module"
}
