#!/bin/sh
# Idempotent first-run provisioning for the local Garage node, via the admin
# API v2 (the Garage image has no shell): single-node layout, the app's
# access key, the bucket, grants, and the bucket CORS policy that browser
# direct uploads need (IMPLEMENTATION_PLAN §4.12).
set -eu

ADMIN="http://storage:3903/v2"
S3="http://storage:3900"
AUTH="Authorization: Bearer ${GARAGE_ADMIN_TOKEN}"
JSON="Content-Type: application/json"

api() { # method path [body]
  if [ $# -ge 3 ]; then curl -sS -f -H "$AUTH" -H "$JSON" -X "$1" "$ADMIN/$2" -d "$3"
  else curl -sS -f -H "$AUTH" -X "$1" "$ADMIN/$2"; fi
}
field() { sed -n "s/.*\"$1\": *\"\{0,1\}\([^\",}]*\)\"\{0,1\}.*/\1/p" | head -1; }

echo "storage-init: waiting for Garage admin API"
i=0; until api GET GetClusterStatus >/dev/null 2>&1; do i=$((i+1)); [ $i -gt 60 ] && exit 1; sleep 1; done

layout_version=$(api GET GetClusterLayout | tr -d '\n' | field version)
if [ "${layout_version:-0}" = "0" ]; then
  node=$(api GET GetClusterStatus | tr -d '\n' | field id)
  api POST UpdateClusterLayout "{\"roles\":[{\"id\":\"$node\",\"zone\":\"dc1\",\"capacity\":1000000000,\"tags\":[]}]}" >/dev/null
  api POST ApplyClusterLayout '{"version":1}' >/dev/null
  echo "storage-init: layout applied"
fi

if ! api GET "GetKeyInfo?id=${S3_ACCESS_KEY_ID}" >/dev/null 2>&1; then
  api POST ImportKey "{\"accessKeyId\":\"${S3_ACCESS_KEY_ID}\",\"secretAccessKey\":\"${S3_SECRET_ACCESS_KEY}\",\"name\":\"opsgrid-app\"}" >/dev/null
  echo "storage-init: access key imported"
fi

bucket_id=$(api GET "GetBucketInfo?globalAlias=${S3_BUCKET}" 2>/dev/null | tr -d '\n' | field id || true)
if [ -z "$bucket_id" ]; then
  bucket_id=$(api POST CreateBucket "{\"globalAlias\":\"${S3_BUCKET}\"}" | tr -d '\n' | field id)
  echo "storage-init: bucket ${S3_BUCKET} created"
fi
api POST AllowBucketKey "{\"bucketId\":\"$bucket_id\",\"accessKeyId\":\"${S3_ACCESS_KEY_ID}\",\"permissions\":{\"read\":true,\"write\":true,\"owner\":true}}" >/dev/null

cors="<CORSConfiguration><CORSRule>"
for o in $(echo "${CORS_ORIGINS}" | tr ',' ' '); do cors="$cors<AllowedOrigin>$o</AllowedOrigin>"; done
cors="$cors<AllowedMethod>PUT</AllowedMethod><AllowedMethod>GET</AllowedMethod><AllowedMethod>HEAD</AllowedMethod>"
cors="$cors<AllowedHeader>content-type</AllowedHeader><AllowedHeader>content-length</AllowedHeader>"
cors="$cors<ExposeHeader>ETag</ExposeHeader><MaxAgeSeconds>600</MaxAgeSeconds></CORSRule></CORSConfiguration>"
md5=$(printf '%s' "$cors" | openssl md5 -binary 2>/dev/null | base64 || true)
curl -sS -f -X PUT "$S3/${S3_BUCKET}?cors" \
  --aws-sigv4 "aws:amz:garage:s3" --user "${S3_ACCESS_KEY_ID}:${S3_SECRET_ACCESS_KEY}" \
  -H "Content-Type: application/xml" ${md5:+-H "Content-MD5: $md5"} --data-binary "$cors" >/dev/null
echo "storage-init: CORS set for ${CORS_ORIGINS}"
echo "storage-init: ready (bucket ${S3_BUCKET})"
