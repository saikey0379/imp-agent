## 依赖
- Go1.21及以上版本

## 构建
### Linux下安装编译环境
1. 登录[golang官网](https://golang.org/dl/)或者[golang中国官方镜像](https://golang.google.cn/dl/)下载最新的稳定版本的go安装包并安装。

   ```bash
   $ wget https://go.dev/dl/go1.21.1.linux-amd64.tar.gz
   # 解压缩后go被安装在/usr/local/go
   $ sudo tar -xzvf ./go1.21.1.linux-amd64.tar.gz -C /usr/local/
   ```

2. 配置go环境变量

   ```bash
   $ cat << "EOF" >> ~/.bashrc
   export GOROOT=/usr/local/go
   export PATH=$PATH:$GOROOT/bin
   EOF
   $ source ~/.bashrc
   ```

### 源代码编译
1. 源码下载与编译

   ```bash
   $ git clone https://github.com/saikey0379/imp-agent.git
   
   $ cd /imp-agent
   $ go build -o ./bin/imp-agent ./cmd/main.go
   ```

   ```bash
   $ ls -l bin
   total 133848
   -rwxr-xr-x  1 root  root    16M  3  1 10:36 imp-agent
   ```
2. RPMbuild

   ```bash
   $ yum -y install rpmbuild
   
   $ VERSION=v0.0.1
   $ tar -zcvf /root/rpmbuild/SOURCES/imp-agent-${VERSION}.tgz bin/ conf/ deploy/systemd/
   $ sed "s/VERSION/${VERSION}/g" deploy/rpmbuild/imp-agent.spec > imp-agent_${VERSION}.spec
   $ rpmbuild -bb imp-agent_${VERSION}.spec
   $ mv /root/rpmbuild/RPMS/x86_64/imp-agent-${VERSION}-0.x86_64.rpm .
   ```
## 安装
1. Agent安装
   ```bash
   $ rpm -ivh /root/rpmbuild/RPMS/x86_64/imp-agent-${VERSION}-0.x86_64.rpm
   ```
2. 启动
   ```bash
   $ systemctl enable imp-agent && systemctl start imp-agent
   ```