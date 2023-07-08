#!/bin/bash

# 仓库名称
repository="labring/endpoints-operator"

# 版本号参数
version=$1

if [[ -z "$version" ]]; then
  # 获取最新release的版本号
  version=$(curl -s "https://api.github.com/repos/$repository/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
fi

# 构建下载链接
download_url="https://github.com/$repository/releases/download/$version/endpoints-operator-${version#v}.tgz"

# 下载指定版本的release
wget $download_url

helm repo index . --url https://github.com/$repository/releases/download/$version
