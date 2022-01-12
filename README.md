# endpoints-operator
> 场景对于外部场景使用固定的endpoint维护增加探活功能

根据service数据生成endpoint数据

1. 根据service的port映射对应的endpoints数据
2. targetPort 不能使用string只能对应int数据



## Usage

```yaml
apiVersion: v1
kind: Service
metadata:
  name: dmz-kube
  namespace: default
  annotations:
    sealyun.com/server: 10.0.112.251 #add this anniotation auto add eps
spec:
  clusterIP: 10.96.0.100
  clusterIPs:
  - 10.96.0.100
  internalTrafficPolicy: Cluster
  ipFamilies:
  - IPv4
  ipFamilyPolicy: SingleStack
  ports:
  - name: https
    port: 6443
    protocol: TCP
    targetPort: 6443
  sessionAffinity: None
  type: ClusterIP
```
