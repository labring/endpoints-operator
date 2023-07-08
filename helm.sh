#!/bin/bash

# 仓库名称
repository="labring/endpoints-operator"

# 获取最新release的版本号
latest_release=$(curl -s "https://api.github.com/repos/$repository/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
# 构建下载链接
download_url="https://github.com/$repository/releases/download/$latest_release/endpoints-operator-${latest_release#v}.tgz"

# 下载最新release
wget $download_url

helm repo index . --url https://github.com/$repository/releases/download/$latest_release
