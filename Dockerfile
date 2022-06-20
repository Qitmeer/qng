FROM ubuntu:18.04
WORKDIR /qng

COPY . /qng

VOLUME ["/qng/logs"]

RUN apt update && apt install build-essential -y && apt install make -y && apt install git -y && apt install wget -y

RUN wget http://mirrors.ustc.edu.cn/golang/go1.18rc1.linux-amd64.tar.gz && \
    tar zxvf go1.18rc1.linux-amd64.tar.gz && mv go /usr/local/ && ln -fs /usr/local/go/bin/* /usr/local/bin/

RUN make

EXPOSE 8131 18131 1234

CMD ["/build/bin/qng"]


