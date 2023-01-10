# cert-manager-webhook-antsdns
cert-manager-webhook-antsdns


### 1 install docker

`curl -fsSL https://get.docker.com/ | sh `  
`sudo systemctl start docker `  
`sudo systemctl start docker.service  `  
`systemctl enable docker  `  



### 2 install kubectl 1.23.8
`cat <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=http://mirrors.aliyun.com/kubernetes/yum/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=0
repo_gpgcheck=0
gpgkey=http://mirrors.aliyun.com/kubernetes/yum/doc/yum-key.gpg http://mirrors.aliyun.com/kubernetes/yum/doc/rpm-package-key.gpg
EOF
yum remove kubeadm kubectl kubelet kubernetes-cni cri-tools socat
yum --showduplicates list kubeadm
yum -y install kubeadm-1.23.8 kubectl-1.23.8 kubelet-1.23.8
systemctl enable kubelet`


### 3 install minikube
`cd /usr/src/cert-manager-antsdns-webhook-release
sudo install minikube-linux-amd64 /usr/local/bin/minikube`


### 4 installcrictl and  init minikube
`cd /usr/src/cert-manager-antsdns-webhook-release
wget https://github.com/kubernetes-sigs/cri-tools/releases/download/$VERSION/crictl-$VERSION-linux-amd64.tar.gz
sudo tar zxvf crictl-$VERSION-linux-amd64.tar.gz -C /usr/local/bin
minikube start --vm-driver=none  --kubernetes-version=v1.23.8 --image-mirror-country=cn --extra-config=kubelet.cgroup-driver=cgroupfs --extra-config=kubeadm.ignore-preflight-errors=SystemVerification --extra-config=apiserver.enable-admission-plugins="NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,Priority,ResourceQuota" --force`
  

### 5 install cert-manager
`cd /usr/src/cert-manager-antsdns-webhook-release
docker load -i quay.io_jetstack_cert-manager-cainjector_1.7.2.tar
docker load -i quay.io_jetstack_cert-manager-controller_1.7.2.tar
docker load -i quay.io_jetstack_cert-manager-webhook_1.7.2.tar
kubectl apply -f  cert-manager-antsdns-webhook-v1/1.7.2cert-manager.yaml`


### 6 install helm
`cd  /usr/src/cert-manager-antsdns-webhook-release
tar zxf helm-v2.17.0-linux-amd64.tar.tar.gz
cd linux-amd64/
mv helm /usr/local/bin/
chmod +x /usr/local/bin/helm`


### 7 install helm tiller
`cd  /usr/src/cert-manager-antsdns-webhook-release
docker load -i docker_ghcr.io.helm.tiller.2_17_0.tar
#docker pull ghcr.io/helm/tiller:v2.17.0
kubectl create serviceaccount --namespace kube-system tiller
kubectl create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
kubectl patch deploy --namespace kube-system tiller-deploy -p '{"spec":{"template":{"spec":{"serviceAccount":"tiller"}}}}'
helm init --service-account tiller --upgrade
helm init --service-account tiller --override spec.selector.matchLabels.'name'='tiller',spec.selector.matchLabels.'app'='helm' --output yaml | sed 's@apiVersion: extensions/v1beta1@apiVersion: apps/v1@' | kubectl apply -f `

 

### 8  install cmctl
`cd /usr/src/cert-manager-antsdns-webhook-release
tar xzf cmctl-linux-amd64.tar.gz
sudo mv cmctl /usr/local/bin
cmctl check api`


### 9 install antsdns_webhook model
`cd /usr/src/cert-manager-antsdns-webhook-release
docker load -i /usr/src/cert-manager-antsdns-webhook-release/docker_image_antsdns-webhook_0.1.1.tar
kubectl apply -f ./pod/00bundle.yaml `


### 10 check env
`docker version
helm version
minikube version
kubectl version
cmctl version 
minikube kubectl -- get pods -A`

---------------------------------------------------------
[root@control-plane src]# docker version
Client: Docker Engine - Community
 Version:           20.10.14
 API version:       1.41
 Go version:        go1.16.15
 Git commit:        a224086
 Built:             Thu Mar 24 01:49:57 2022
 OS/Arch:           linux/amd64
 Context:           default
 Experimental:      true

Server: Docker Engine - Community
 Engine:
  Version:          20.10.14
  API version:      1.41 (minimum version 1.12)
  Go version:       go1.16.15
  Git commit:       87a90dc
  Built:            Thu Mar 24 01:48:24 2022
  OS/Arch:          linux/amd64
  Experimental:     false
 containerd:
  Version:          1.5.11
  GitCommit:        3df54a852345ae127d1fa3092b95168e4a88e2f8
 runc:
  Version:          1.0.3
  GitCommit:        v1.0.3-0-gf46b6ba
 docker-init:
  Version:          0.19.0
  GitCommit:        de40ad0
[root@control-plane src]# helm version
Client: &version.Version{SemVer:"v2.17.0", GitCommit:"a690bad98af45b015bd3da1a41f6218b1a451dbe", GitTreeState:"clean"}
Server: &version.Version{SemVer:"v2.17.0", GitCommit:"a690bad98af45b015bd3da1a41f6218b1a451dbe", GitTreeState:"clean"}
[root@control-plane src]# minikube version
minikube version: v1.28.0
commit: 986b1ebd987211ed16f8cc10aed7d2c42fc8392f
[root@control-plane src]# kubectl version
Client Version: version.Info{Major:"1", Minor:"23", GitVersion:"v1.23.8", GitCommit:"a12b886b1da059e0190c54d09c5eab5219dd7acf", GitTreeState:"clean", BuildDate:"2022-06-16T05:57:43Z", GoVersion:"go1.17.11", Compiler:"gc", Platform:"linux/amd64"}
Server Version: version.Info{Major:"1", Minor:"23", GitVersion:"v1.23.8", GitCommit:"a12b886b1da059e0190c54d09c5eab5219dd7acf", GitTreeState:"clean", BuildDate:"2022-06-16T05:51:36Z", GoVersion:"go1.17.11", Compiler:"gc", Platform:"linux/amd64"}
[root@control-plane src]# cmctl version 
Client Version: util.Version{GitVersion:"v1.10.1", GitCommit:"a96bae172ddb1fcd4b57f1859ab9d1a9e94f7451", GitTreeState:"", GoVersion:"go1.19.3", Compiler:"gc", Platform:"linux/amd64"}
Server Version: &versionchecker.Version{Detected:"v1.7.2", Sources:map[string]string{"crdLabelVersion":"v1.7.2"}}
[root@control-plane src]# minikube kubectl -- get pods -A
NAMESPACE      NAME                                                      READY   STATUS    RESTARTS       AGE
cert-manager   antsdns-webhook-57c74d9795-sqlxb                          1/1     Running   0              17h
cert-manager   cert-manager-798ff58464-dwzrh                             1/1     Running   2 (17h ago)    17h
cert-manager   cert-manager-cainjector-5779577c5f-x6g68                  1/1     Running   1 (17h ago)    17h
cert-manager   cert-manager-webhook-5fbbdccffd-bq9zj                     1/1     Running   1 (17h ago)    17h
kube-system    coredns-65c54cc984-dl62s                                  1/1     Running   8 (17h ago)    46h
kube-system    etcd-control-plane.minikube.internal                      1/1     Running   10 (17h ago)   46h
kube-system    kube-apiserver-control-plane.minikube.internal            1/1     Running   5 (17h ago)    46h
kube-system    kube-controller-manager-control-plane.minikube.internal   1/1     Running   11 (17h ago)   46h
kube-system    kube-proxy-lv7zf                                          1/1     Running   6 (17h ago)    46h
kube-system    kube-scheduler-control-plane.minikube.internal            1/1     Running   10 (17h ago)   46h
kube-system    storage-provisioner                                       1/1     Running   11 (17h ago)   46h
kube-system    tiller-deploy-74bcf4c66c-fhv5b                            1/1     Running   5 (17h ago)    46h
[root@control-plane src]# 


### 11 creat cecret clusterissuer
`cd /usr/src/cert-manager-antsdns-webhook-release
kubectl apply -f ./prod/01cecret.yaml
kubectl apply -f ./prod/02letsencrypt-clusterissuer.yaml`
###
01cecret.yaml =====>appId: xxxx,appKey: xxxx
###
02letsencrypt-clusterissuer =====>ispAddress: "xxxx"


### 12 apply cert
`cd /usr/src/cert-manager-antsdns-webhook-release
kubectl apply -f ./prod/03certif-165668-com-clusterissuer.yaml`


### 13 check cert status
`cmctl status certificate 165668-com  -n cert-manager`


### 14 view cert
`kubectl get secret 165668-com-tls -n cert-manager -o yaml`

