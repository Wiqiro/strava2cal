#!/bin/sh
set -e

echo "Starting Go server..."
/usr/local/bin/server &

echo "Starting nginx..."
nginx -g "daemon off;"