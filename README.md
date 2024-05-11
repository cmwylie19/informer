```bash
k3d cluster delete --all
k3d cluster create
docker build -t cmwylie19/informer:v1alpha1 .  
k3d image import informer:v1alpha1 -c k3s-default
k apply -f manifests.yaml
sleep 10;
k logs -n informer -l app=informer -f 
k run i --image=nginx
```



Working:
- [x] m3 
- [x] m2
- [x] main.go


TODO:
- [ ] Delete a pod when a pod is created
- [ ] Deploy Istio
- [ ] Deploy CronJob

