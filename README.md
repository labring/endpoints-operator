# endpoints-operator

> 对于集群内访问集群外部服务场景使用固定的endpoint维护增加探活功能

### 注意事项:

- v0.1.1 版本的数据是sealyun.com的domain
- v0.2.0 之后所有的domain都是sealos.io
- v0.2.1 调整了Hosts配置，升级需要注意一下

也可以手动执行一下脚本,namespace为xxx
```shell
for cep in $(kubectl get cep -n xxx  -o jsonpath={.items[*].metadata.name});do kubectl patch cep -n xxx --type='json' -p='[{"op": "replace", "path": "/metadata/finalizers", "value":[]}]'  $cep;done
```

升级资源之前最好先备份一下cr删除之后再重新创建即可。

## 背景

- 在实际使用中，两个K8s集群之间的服务经常有互相访问和访问集群外部某些服务的需求，通常的解决方案为手动维护固定的Services和Endpoints或者直接在业务配置中写死IP，在这时候，是没有对外部服务进行探活的功能的，无法做到高可用。如果需要高可用一般是引入外部高可用LB来解决，但这样增加了复杂度，且好多公司不具备引入条件，不是最优解决方案。
- 众所周知，Kube-Proxy的主要功能是维护集群内Services和Endpoints并在对应主机上创建对应的IPVS规则。从而达到我们可以在Pod里面通过ClusterIP访问的目的。

由此，新的想法诞生了: 写一个controller，维护一个CRD来自动创建需要访问的外部服务对应的Service和Endpoint，并对创建的Endpoint中的外部服务数据（IP:PORT列表）进行探活，探活失败则移除对应的外部服务数据。

## 介绍

endpoints-operator是一个云原生、高可靠性、高性能、面向K8s内部服务访问外部服务的具备探活功能的4层LB。

### 特性

- 更加贴近云原生
- 声明式API：探活的定义方式与Kubelet保持一致，还是熟悉的语法、熟悉的味道
- 高可靠性：原生Service、Endpoint资源，拒绝重复造轮子
- 高性能、高稳定：原生IPVS高性能4层负载均衡

  

### 核心优势

- 完全使用K8s原生的Service、Endpoint资源，无自定义IPVS策略，依托K8s的Service能力，高可靠。
- 通过controller管理一个CRD资源ClusterEndpoint（缩写cep）即可，无需手动管理Service和Endpoint两个资源
- 完全兼容已有的自定义Service、Endpoint资源，可无缝切换至endpoints-operator管理。
- 原生的IPVS 4层负载，未引入Nginx、HAProxy等LB，降低了复杂度，满足高性能和高稳定性的需求

### 使用场景

主要使用在集群内部的Pod需要访问外部服务的场景，比如数据库、中间件等，通过endpoints-operator的探活能力，可及时将有问题的后端服务剔除，避免受单个宕机副本影响，并可查看status获取后端服务健康状态和探活失败的服务。

## helm 安装

```bash
VERSION="0.2.1"
wget https://github.com/labring/endpoints-operator/releases/download/v${VERSION}/endpoints-operator-${VERSION}.tgz
helm install -n kube-system endpoints-operator ./endpoints-operator-${VERSION}.tgz
```

## sealos 安装

```bash
sealos run labring/endpoints-operator:v0.2.1
```

## Usage

```yaml
apiVersion: sealos.io/v1beta1
kind: ClusterEndpoint
metadata:
  name: wordpress
  namespace: default
spec:
  periodSeconds: 10
  ports:
    - name: wp-https
      hosts:
        ## 端口相同的hosts
        - 10.33.40.151
        - 10.33.40.152
      protocol: TCP
      port: 38081
      targetPort: 443
      tcpSocket:
        enable: true
      timeoutSeconds: 1
      failureThreshold: 3
      successThreshold: 1
    - name: wp-http
      hosts:
        ## 端口相同的hosts
        - 10.33.40.151
        - 10.33.40.152
      protocol: TCP
      port: 38082
      targetPort: 80
      httpGet:
        path: /healthz
        scheme: http
      timeoutSeconds: 1
      failureThreshold: 3
      successThreshold: 1      
    - name: wp-udp
      hosts:
        ## 端口相同的hosts
        - 10.33.40.151
        - 10.33.40.152
      protocol: UDP
      port: 38003
      targetPort: 1234
      udpSocket:
        enable: true
        data: "This is flag data for UDP svc test"
      timeoutSeconds: 1
      failureThreshold: 3
      successThreshold: 1
    - name: wp-grpc
      hosts:
        ## 端口相同的hosts
        - 10.33.40.151
        - 10.33.40.152
      protocol: TCP
      port: 38083
      targetPort: 8080
      grpc:
        enable: true
      timeoutSeconds: 1
      failureThreshold: 3
      successThreshold: 1
```

## 总结
"endpoints-operator” 的引入，对产品无侵入以及云原生等特性解决了在集群内部访问外部服务等问题。这个思路将会成为以后开发或者运维的标配，也是一个比较完善的项目，从开发的角度换个思路更优雅的去解决一些问题。
