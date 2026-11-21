#!/bin/sh
# Creates the AWS infrastructure in MiniStack. Idempotent: safe to run several times.
# Resources: S3 bucket, 3 RDS Postgres databases (one per service) and an ElastiCache (Redis) cluster.
set -e

BUCKET="${S3_BUCKET:-futhub-photos}"

status() {
  aws rds describe-db-instances --db-instance-identifier "$1" \
    --query 'DBInstances[0].DBInstanceStatus' --output text 2>/dev/null || echo "none"
}

# create_db <service>  ->  instance futhub-<service>, database db_<service>
create_db() {
  name="futhub-$1"
  for attempt in 1 2 3; do
    s=$(status "$name")
    if [ "$s" = "failed" ]; then
      echo "   $name failed, recreating (attempt $attempt)"
      aws rds delete-db-instance --db-instance-identifier "$name" --skip-final-snapshot >/dev/null 2>&1 || true
      sleep 2; s="none"
    fi
    if [ "$s" = "none" ]; then
      aws rds create-db-instance \
        --db-instance-identifier "$name" \
        --db-instance-class db.t3.micro \
        --engine postgres \
        --master-username "$DB_USER" \
        --master-user-password "$DB_PASSWORD" \
        --db-name "db_$1" \
        --allocated-storage 20 >/dev/null
    fi
    # wait until available (the first run pulls the Postgres image, which can take a while)
    i=0
    while [ $i -lt 60 ]; do
      s=$(status "$name")
      [ "$s" = "available" ] && { echo "   $name available"; return 0; }
      [ "$s" = "failed" ] && break
      sleep 3; i=$((i+1))
    done
  done
  echo "ERROR: $name did not become available" >&2
  return 1
}

echo ">> S3 (bucket $BUCKET)"
aws s3api create-bucket --bucket "$BUCKET" >/dev/null 2>&1 || echo "   (bucket already exists)"

echo ">> RDS (one Postgres per service)"
for svc in main payment notification; do create_db "$svc"; done

echo ">> ElastiCache (Redis)"
aws elasticache create-cache-cluster \
  --cache-cluster-id futhub-cache \
  --engine redis \
  --cache-node-type cache.t3.micro \
  --num-cache-nodes 1 >/dev/null 2>&1 || echo "   (futhub-cache already exists)"

echo ">> Endpoints (as seen from inside the Docker network):"
aws rds describe-db-instances --query 'DBInstances[].[DBInstanceIdentifier,Endpoint.Address,Endpoint.Port]' --output text
aws elasticache describe-cache-clusters --show-cache-node-info \
  --query 'CacheClusters[].[CacheClusterId,CacheNodes[0].Endpoint.Address,CacheNodes[0].Endpoint.Port]' --output text
echo ">> init finished"
