FROM carltraveler/witness_runtimev0:v1

MAINTAINER steven chenglin.cn@163.com

ADD images /images

WORKDIR /images

ENV PATH /root/.cargo/bin:$PATH

EXPOSE 8080

CMD ["bash", "run_server.bash"]
