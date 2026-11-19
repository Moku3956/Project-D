# EC2を毎日決まった時刻に停止・起動する。
# EventBridge Schedulerのユニバーサルターゲットで EC2 API を直接呼ぶため、Lambdaは使わない。

locals {
  ec2_stop_schedule  = "cron(0 21 * * ? *)" # 毎日 21:00 に停止
  ec2_start_schedule = "cron(0 9 * * ? *)"  # 毎日 9:00 に起動
}

resource "aws_iam_role" "scheduler" {
  name = "mokudb-scheduler-production-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "scheduler.amazonaws.com" }
      Action    = "sts:AssumeRole"
      Condition = {
        StringEquals = { "aws:SourceAccount" = "660941537209" }
      }
    }]
  })
}

resource "aws_iam_role_policy" "scheduler" {
  name = "mokudb-scheduler-production-rolePolicy"
  role = aws_iam_role.scheduler.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid      = "StartStopInstance"
      Effect   = "Allow"
      Action   = ["ec2:StartInstances", "ec2:StopInstances"]
      Resource = aws_instance.this.arn
    }]
  })
}

resource "aws_scheduler_schedule" "ec2_stop" {
  name                         = "mokudb-ec2-stop-production"
  schedule_expression          = local.ec2_stop_schedule
  schedule_expression_timezone = "Asia/Tokyo"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = "arn:aws:scheduler:::aws-sdk:ec2:stopInstances"
    role_arn = aws_iam_role.scheduler.arn
    input    = jsonencode({ InstanceIds = [aws_instance.this.id] })
  }
}

resource "aws_scheduler_schedule" "ec2_start" {
  name                         = "mokudb-ec2-start-production"
  schedule_expression          = local.ec2_start_schedule
  schedule_expression_timezone = "Asia/Tokyo"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = "arn:aws:scheduler:::aws-sdk:ec2:startInstances"
    role_arn = aws_iam_role.scheduler.arn
    input    = jsonencode({ InstanceIds = [aws_instance.this.id] })
  }
}
