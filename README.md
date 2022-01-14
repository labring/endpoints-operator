# endpoints-operator

> 对于集群内访问集群外部服务场景使用固定的endpoint维护增加探活功能

## 背景

在实际使用中，两个K8s集群内的服务经常有互相访问和访问集群外部某些服务的需求，通常的解决方案为写固定的servcie和edpoints或者直接写IP等来解决，在这时候，是没有对外部服务的探活功能的，如果需要探活功能一般是引入一个高可用LB来解决。

本项目利用了K8s原生service功能的ipvs虚IP和负载功能，保证了虚IP本身是高可用的，同时增加了对后端endpoints的定时探活功能，可以在后端endpoints探活失败后从endpoints列表中踢出，保证了svc后端的epdpoints永远是健康的。

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
```
