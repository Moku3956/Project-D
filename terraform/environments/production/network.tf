resource "aws_vpc" "this" {
  cidr_block = "10.16.0.0/20"

  tags = {
    Name = "mokudb-prod-vpc"
  }
}

resource "aws_subnet" "public" {
  vpc_id                  = aws_vpc.this.id
  cidr_block              = "10.16.0.0/24"
  availability_zone       = "ap-northeast-3a"
  map_public_ip_on_launch = false

  tags = {
    Name = "mokudb-prod-public-subnet-1"
  }
}

resource "aws_internet_gateway" "this" {
  vpc_id = aws_vpc.this.id

  tags = {
    Name = "mokudb-prod-igw"
  }
}

resource "aws_route_table" "this" {
  vpc_id = aws_vpc.this.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.this.id
  }

  tags = {}
}

resource "aws_security_group" "ec2" {
  name        = "mokudb-ec2-sg-production"
  description = "sg for production instances"
  vpc_id      = aws_vpc.this.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Environment = "production"
  }
}
