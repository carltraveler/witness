FROM ubuntu:18.04

MAINTAINER steven chenglin.cn@163.com                                                                                                                     

ADD deploy_run.bash /repodir/
ADD wallet.dat /repodir/

RUN apt update
RUN apt install -y curl
RUN curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain nightly 
RUN export PATH="$HOME/.cargo/bin:$PATH";rustup target add wasm32-unknown-unknown
ADD ontio-wasm-build /bin

WORKDIR /repodir

ENV PATH /repodir:$PATH

EXPOSE 8080

CMD ["bash", "deploy_run.bash"]
