terraform {
  required_version = "~> 1.5"
  required_providers {
    http-client = {
      source  = "dmachard/http-client"
      version = "0.3.0"
    }
  }
}




check "health" {
  data "http" "terraform_io" {
    url = "http://10.1.0.100:9100/metrics"
  }
  assert {
    condition = data.http.terraform_io.status_code == 201
    error_message = "Status check failed"
  }

}