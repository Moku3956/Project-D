resource "aws_instance" "this" {
  ami                    = "ami-0419b2b3561045ac7"
  instance_type          = "t3.nano"
  subnet_id              = aws_subnet.public.id
  vpc_security_group_ids = [aws_security_group.ec2.id]
  iam_instance_profile   = "mokudb-ec2-role"

  tags = {
    Name = "mokudb-stg"
  }
}

resource "aws_eip" "this" {
  domain   = "vpc"
  instance = aws_instance.this.id
}
