# endpoints-operator
> 场景对于外部场景使用固定的endpoint维护增加探活功能

根据service数据生成endpoint数据

1. 根据service的port映射对应的endpoints数据
2. targetPort 不能使用string只能对应int数据



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
