# ChangeLog

## [v0.1.0](#v0.1.0)

- 设计完成CRD `ClusterEndpoint` 来维护service和endpoint功能
- 完成controller的探活功能，支持HTTP和TCP Port，使用controller维护CRD的status数据
- 完善部署的helm chart

## [v0.1.1](#v0.1.1)

- 支持GRPC功能
- 支持UDP功能
- 新增对应的监控数据
- 新增cepctl工具做数据转换
- 新增webhook证书维护代码

## [v0.2.0](#v0.2.0)

- 所有的domain更正为sealos.io
- FixBug #86 不支持clusterIP=None

## [v0.2.1](#v0.2.1)

- FixBug #103  endpoint 无法更新的问题

