FROM ubuntu:18.04

MAINTAINER steven chenglin.cn@163.com                                                                                                                     

ADD config_run.bash /repodir/

RUN apt update
RUN apt install -y git
RUN apt install -y golang

WORKDIR /repodir

ENV PATH /repodir:$PATH

EXPOSE 8080

CMD ["bash", "config_run.bash"]
