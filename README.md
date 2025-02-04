# S3 Image Optimizer - AWS Lambda (Go + Terraform)

## 📌 Overview
This project is an **AWS Lambda function** written in **Go** that automatically:
✅ Fetches images from an S3 bucket.  
✅ Optimizes (resizes & compresses) images using `imaging`.  
✅ Uploads optimized images to another S3 bucket.  
✅ Uses a worker pool for efficient parallel processing.  

**Infrastructure** is deployed using **Terraform**, ensuring:
- IAM roles & permissions for Lambda & S3.
- Automatic S3 event triggers (optional) to process new uploads.

---

## 🛠️ Technologies Used

### Backend
- **Go** (AWS Lambda runtime)
- **AWS SDK for Go v2**
- **Disintegration/imaging** (Image processing)

### Infrastructure
- **Terraform** (Infrastructure as Code)
- **AWS Lambda** (Serverless compute)
- **Amazon S3** (Storage for images)
- **IAM** (Permissions & security)

---

## 🚀 Setup & Deployment

### 1️⃣ Build & Package Lambda
Ensure you have Go installed, then build and zip the Lambda function:
```sh
GOOS=linux GOARCH=amd64 go build -o bootstrap main.go
zip lambda.zip bootstrap
```

### 2️⃣ Deploy with Terraform
```sh
terraform init
terraform apply -auto-approve
```

### 3️⃣ Upload an Image to S3
```sh
aws s3 cp image.jpg s3://my-source-bucket/uuid/image.jpg
```

### 4️⃣ Check Optimized Image in Destination Bucket
```sh
aws s3 ls s3://my-destination-bucket/optimized/uuid/
```

---

## 🔧 Configuration
Modify environment variables in **Terraform (`main.tf`)**:
```hcl
environment {
  variables = {
    SOURCE_BUCKET      = "my-source-bucket"
    DESTINATION_BUCKET = "my-destination-bucket"
  }
}
```

---

## 🛠 Troubleshooting
- Check **CloudWatch logs** for errors:
  ```sh
  aws logs tail /aws/lambda/ImageOptimizerLambda --follow
  ```
- Ensure IAM permissions allow S3 read/write access.

---

## 📜 License
This project is licensed under the MIT License.
