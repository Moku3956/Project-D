resource "aws_iam_role" "deploy" {
  name = "mokudb-deploy-staging-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = "arn:aws:iam::660941537209:oidc-provider/token.actions.githubusercontent.com" }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = { "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com" }
        StringLike   = { "token.actions.githubusercontent.com:sub" = "repo:Moku3956@218535046/Project-D@1303751823:ref:refs/heads/stage" }
      }
    }]
  })
}

resource "aws_iam_role_policy" "deploy" {
  name = "mokudb-deploy-staging-rolePolicy"
  role = aws_iam_role.deploy.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "S3Frontend"
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:DeleteObject", "s3:ListBucket"]
        Resource = ["arn:aws:s3:::mokudb-frontend-stg", "arn:aws:s3:::mokudb-frontend-stg/*"]
      },
      {
        Sid      = "S3Artifacts"
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:ListBucket"]
        Resource = ["arn:aws:s3:::mokudb-artifacts-stg", "arn:aws:s3:::mokudb-artifacts-stg/*"]
      },
      {
        Sid      = "CloudFrontInvalidation"
        Effect   = "Allow"
        Action   = "cloudfront:CreateInvalidation"
        Resource = "*"
      },
      {
        Sid      = "SSMSendCommandInstance"
        Effect   = "Allow"
        Action   = "ssm:SendCommand"
        Resource = "arn:aws:ec2:ap-northeast-3:660941537209:instance/*"
        Condition = {
          StringEquals = { "ssm:resourceTag/Environment" = "staging" }
        }
      },
      {
        Sid      = "SSMSendCommandDocument"
        Effect   = "Allow"
        Action   = "ssm:SendCommand"
        Resource = "arn:aws:ssm:ap-northeast-3::document/AWS-RunShellScript"
      },
      {
        Sid      = "SSMGetCommandInvocation"
        Effect   = "Allow"
        Action   = ["ssm:GetCommandInvocation", "ssm:ListCommandInvocations"]
        Resource = "*"
      }
    ]
  })
}
