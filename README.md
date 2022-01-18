# endpoints-operator

> 对于集群内访问集群外部服务场景使用固定的endpoint维护增加探活功能

## 背景

在实际使用中，两个K8s集群之间的服务经常有互相访问和访问集群外部某些服务的需求，通常的解决方案为写固定的servcie和edpoints或者直接写IP，在这时候，是没有对外部服务的探活功能的，无法做到高可用。如果需要高可用一般是引入外部高可用LB来解决。

本项目利用了K8s原生service功能的ipvs虚IP和负载功能，保证了虚IP本身是高可用的，同时增加了对后端endpoints的定时探活功能，可以在后端endpoints探活失败后从endpoints列表中踢出，保证了svc后端的epdpoints永远是健康的。

## 介绍

endpoints-operator是一个云原生、高可靠性、高性能、面向K8s内部服务访问外部服务的具备探活功能的4层LB

### 特性

- 云原生

- 声明式管理：Servcie和kubelet探活的定义方式，还是熟悉的语法、熟悉的味道

- 高可靠性：原生Service、Endpoint资源，拒绝重复造轮子

- 高性能、高稳定：原生IPVS高性能4层负载均衡

  

### 核心优势

完全使用K8s原生的Service、Endpoint资源，无自定义ipvs策略，依托K8s的service能力，高可靠。

完全兼容已有的自定义service、endpoint资源，可无缝切换至endpoints-operator管理。

声明式管理

通过CRD、controller可管理一个资源即可，无需手动管理service和endpoint两个资源

原生的ipvs4层负载，未引入ingress、nginx、haproxy等LB，满足高性能、和高稳定性的需求

### 使用场景

主要使用在集群内部的pod需要访问外边服务的场景，比如数据库、中间件等，通过endpoints-operator的探活能力，可及时将有问题的后端服务剔除，避免受单个宕机副本影响，并可查看status获取后端服务健康状态和探活失败的服务。

## 安装

```bash
helm install -n <namespce> endpoints-operator config/charts/endpoints-operator
```

## Usage

```yaml
apiVersion: sealyun.com/v1beta1
kind: ClusterEndpoint
metadata:
  name: dmz-kube
spec:
  clusterIP: 10.96.0.100
  periodSeconds: 10
  hosts:
    - 10.0.112.255
  ports:
    - name: https
      port: 6443
      protocol: TCP
      targetPort: 6443
      timeoutSeconds: 1
      successThreshold: 1
      failureThreshold: 3
      tcpSocket:
        enable: true
    - name: http
      port: 80
      protocol: TCP
      targetPort: 80
      httpGet:
        path: /
        scheme: http
    - name: udp
      port: 80
      protocol: UDP
      targetPort: 80
      ###udp暂时不支持探活
```