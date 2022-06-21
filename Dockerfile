FROM ubuntu:18.04
WORKDIR /qng

COPY . /qng

VOLUME ["/qng/logs"]

#RUN sed -i s@/archive.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list && \
#    sed -i s@/security.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list && apt update clean

RUN apt update && apt install build-essential -y && apt install make -y && apt install git -y && apt install wget -y

RUN wget http://mirrors.ustc.edu.cn/golang/go1.18rc1.linux-amd64.tar.gz && \
    tar zxvf go1.18rc1.linux-amd64.tar.gz && mv go /usr/local/ && ln -fs /usr/local/go/bin/* /usr/local/bin/

RUN make

CMD ["./build/bin/qng"]

