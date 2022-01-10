
crd 使用方法

```go
    rootCtx := context.Background()
	log := r.Log.WithValues("deployment", req.NamespacedName)
	ctx := context.WithValue(rootCtx, helper.LoggerApplicationCtxKey, log)

	// your logic here
	deployment := &appsv1.Deployment{}
	ctr := controller.Controller{
		Client:   r.Client,
		Eventer:  r.Eventer,
		Operator: r,
		Gvk: schema.GroupVersionKind{
			Group:   appsv1.SchemeGroupVersion.Group,
			Version: appsv1.SchemeGroupVersion.Version,
			Kind:    "Deployment",
		},
        LoggerKey: "logger",
        Module: "application",
	}
	deployment.APIVersion = ctr.Gvk.GroupVersion().String()
	deployment.Kind = ctr.Gvk.Kind
	return ctr.Run(ctx, req, deployment)
```

