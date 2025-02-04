provider "aws" {
  region = "us-east-1" # Change as needed
}

# S3 Buckets
resource "aws_s3_bucket" "source_bucket" {
  bucket = "my-source-bucket-12345"
}

resource "aws_s3_bucket" "destination_bucket" {
  bucket = "my-destination-bucket-12345"
}

# IAM Role for Lambda
resource "aws_iam_role" "lambda_role" {
  name = "lambda_s3_image_optimizer_role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow"
    }
  ]
}
EOF
}

# IAM Policy: Allow Lambda to Read/Write to S3
resource "aws_iam_policy" "lambda_s3_policy" {
  name        = "LambdaS3Policy"
  description = "Allows Lambda to read from source S3 and write to destination S3"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "${aws_s3_bucket.source_bucket.arn}",
        "${aws_s3_bucket.source_bucket.arn}/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject"
      ],
      "Resource": [
        "${aws_s3_bucket.destination_bucket.arn}",
        "${aws_s3_bucket.destination_bucket.arn}/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:*"
    }
  ]
}
EOF
}

# Attach Policy to Role
resource "aws_iam_role_policy_attachment" "lambda_s3_attachment" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = aws_iam_policy.lambda_s3_policy.arn
}

# Lambda Function
resource "aws_lambda_function" "image_optimizer" {
  function_name = "ImageOptimizerLambda"
  role          = aws_iam_role.lambda_role.arn

  runtime       = "go1.x"
  handler       = "bootstrap"  # Go binary name
  filename      = "lambda.zip" # Ensure this file exists (build the Go app & zip it)

  timeout = 30 # Adjust based on image size

  environment {
    variables = {
      SOURCE_BUCKET      = aws_s3_bucket.source_bucket.bucket
      DESTINATION_BUCKET = aws_s3_bucket.destination_bucket.bucket
    }
  }
}

# (Optional) S3 Event Trigger: Run Lambda when a new image is uploaded
resource "aws_s3_bucket_notification" "s3_event_trigger" {
  bucket = aws_s3_bucket.source_bucket.id

  lambda_function {
    lambda_function_arn = aws_lambda_function.image_optimizer.arn
    events              = ["s3:ObjectCreated:*"]
  }
}

# Allow S3 to invoke Lambda
resource "aws_lambda_permission" "allow_s3_trigger" {
  statement_id  = "AllowExecutionFromS3"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.image_optimizer.function_name
  principal     = "s3.amazonaws.com"
  source_arn    = aws_s3_bucket.source_bucket.arn
}
