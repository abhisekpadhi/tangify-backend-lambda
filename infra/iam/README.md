# Terraform Infrastructure for Tangify Backend Lambda

This directory contains Terraform configuration to create the IAM role required by the Lambda function.

## Resources Created

- **IAM Role**: `tangify-backend-lambda-role` with trust policy for Lambda service
- **Inline Policy**: Grants permissions for:
  - Read SSM parameter `tangify.jwt.secret`
  - Read/write access to all DynamoDB tables
  - Write access to CloudWatch logs for the Lambda function

## Usage

1. Initialize Terraform:
   ```bash
   terraform init
   ```

2. Review the plan:
   ```bash
   terraform plan
   ```

3. Apply the configuration:
   ```bash
   terraform apply
   ```

4. To use the role ARN in your SAM template, you can reference the output:
   ```bash
   terraform output lambda_role_arn
   ```

## Variables

- `environment` (default: "dev"): Environment name for tagging resources

## Outputs

- `lambda_role_arn`: ARN of the IAM role
- `lambda_role_name`: Name of the IAM role

