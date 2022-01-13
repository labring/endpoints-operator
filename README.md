# endpoints-operator
> 场景对于外部场景使用固定的endpoint维护增加探活功能

## 背景
在很多场景下，用户集群可能需要访问集群外的数据，这时候又想使用service提供的功能，我们一般创建一个空的service，并手动绑定对应的endpoint.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: mysql-kube
  namespace: default
spec:
  clusterIP: 10.96.0.100
  clusterIPs:
  - 10.96.0.100
  internalTrafficPolicy: Cluster
  ipFamilies:
  - IPv4
  ipFamilyPolicy: SingleStack
  ports:
  - name: mysql
    port: 3306
    protocol: TCP
    targetPort: 3306
  sessionAffinity: None
  type: ClusterIP
---
apiVersion: v1
kind: Endpoints
metadata:
  name: mysql-kube
  namespace: default
subsets:
- addresses:
  - ip: 10.0.99.251
  ports:
  - name: mysql
    port: 3306
    protocol: TCP
```

这样手动设置了对应的

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
