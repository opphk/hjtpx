#!/bin/bash

cd /workspace/hjtpx/backend

echo "Removing redundant database files with duplicate declarations..."

git rm -f pkg/database/advanced_optimizer.go \
   pkg/database/connection_pool_optimizer.go \
   pkg/database/data_archiving.go \
   pkg/database/enhanced_pool_optimizer.go \
   pkg/database/pool_metrics.go \
   pkg/database/slow_query_analyzer.go

echo "Files removed. Attempting to build..."

go build -o hjtpx ./cmd/api/main.go 2>&1 | head -20
