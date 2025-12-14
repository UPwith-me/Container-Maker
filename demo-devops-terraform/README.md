# Demo: DevOps with Terraform & Kubernetes

This demo showcases Container-Maker for DevOps workflows using Terraform and kubectl.

## Quick Start

```bash
# Initialize the dev container
cm shell

# Initialize Terraform
terraform init

# Plan infrastructure
terraform plan

# Apply changes
terraform apply
```

## Features Demonstrated

- **Terraform**: Infrastructure as Code
- **kubectl**: Kubernetes CLI
- **Helm**: Package manager for K8s
- **AWS CLI**: Cloud provider tools

## Tools Included

| Tool | Version | Purpose |
|------|---------|---------|
| Terraform | 1.6+ | IaC |
| kubectl | 1.28+ | K8s management |
| Helm | 3.x | K8s packages |
| aws-cli | 2.x | AWS management |

## Project Structure

```
demo-devops-terraform/
├── .devcontainer/
│   └── devcontainer.json
├── main.tf
├── variables.tf
├── outputs.tf
└── README.md
```

## Learn More

- [Terraform Docs](https://terraform.io/docs)
- [Container-Maker Guide](../README.md)
