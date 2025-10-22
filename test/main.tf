resource "local_file" "test_file" {
  filename = "test_file.txt"
  content  = "this is a test\n"
}

module "foo" {
  source = "./modules/foo"
}

module "gcloud" {
  source  = "terraform-google-modules/gcloud/google"
  version = "~> 4.0"
}
