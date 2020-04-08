FROM ubuntu:18.04

MAINTAINER steven chenglin.cn@163.com                                                                                                                     

ADD deploy_run.bash /repodir/

WORKDIR /repodir

ENV PATH /repodir:$PATH

EXPOSE 8080

CMD ["bash", "deploy_run.bash"]
