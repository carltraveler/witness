FROM ubuntu:18.04

MAINTAINER steven chenglin.cn@163.com                                                                                                                     

ADD config_server /serverdir/

WORKDIR /serverdir

ENV PATH /serverdir:$PATH

EXPOSE 8080

CMD ["config_server"]
