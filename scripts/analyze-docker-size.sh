#!/bin/bash

echo "Docker镜像大小分析"
echo "================================"

# 分析每个镜像的层
echo ""
echo "镜像层大小分析:"
for image in $(docker images --format "{{.Repository}}:{{.Tag}}" | grep hjtpx); do
  echo ""
  echo "镜像: $image"
  docker history "$image" --no-trunc --format "{{.Size}}\t{{.CreatedBy}}" | \
    awk '{size=$1; cmd=$2; if (size ~ /MB/) {mb+=substr(size,1,length(size)-2)} else if (size ~ /KB/) {kb+=substr(size,1,length(size)-2)/1024}} END {printf "  总大小: %.2f MB\n", mb+kb}'
done

echo ""
echo "最大的层:"
docker images | grep hjtpx | head -1 | awk '{print $3}' | xargs -I {} docker history {} --no-trunc --format "{{.Size}}\t{{.CreatedBy}}" | sort -hr | head -10

echo ""
echo "优化建议:"
echo "- 使用多阶段构建分离构建和运行环境"
echo "- 合并RUN指令减少层数"
echo "- 使用.dockerignore排除不必要的文件"
echo "- 利用构建缓存优化构建速度"
echo "- 考虑使用更小的基础镜像(alpine)"
