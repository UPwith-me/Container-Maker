# Demo Terraform Configuration
# This creates a simple AWS S3 bucket for demonstration

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

# Create an S3 bucket
resource "aws_s3_bucket" "demo" {
  bucket = var.bucket_name

  tags = {
    Name        = "Container-Maker Demo"
    Environment = "Dev"
    ManagedBy   = "Terraform"
  }
}

# Enable versioning
resource "aws_s3_bucket_versioning" "demo" {
  bucket = aws_s3_bucket.demo.id
  versioning_configuration {
    status = "Enabled"
  }
}
