FROM centos:centos7

WORKDIR /root

RUN yum -y update || true &&\
    yum -y install epel-release &&\
    yum -y install zlib-devel &&\
    yum -y install wget &&\
    yum -y install gcc &&\
    yum -y install gcc-c++ &&\
    yum -y install autogen &&\
    yum -y install libtool &&\
    yum -y install make &&\
    yum -y install autoconf &&\
    yum -y install automake &&\
    yum -y install freetype-devel &&\
    yum -y install SDL-devel &&\
    yum -y install python-devel &&\
    yum -y install python-pip &&\
    yum -y install bzip2 &&\
    yum -y install swig &&\
    yum -y install openssl-devel &&\
    yum -y install gcc-gfortran &&\
    curl -L https://repo.continuum.io/miniconda/Miniconda2-4.3.21-Linux-x86_64.sh -o miniconda.sh &&\
    /usr/bin/bash miniconda.sh -b -p $HOME/miniconda2 &&\
    rm miniconda.sh &&\
    export PATH="$HOME/miniconda2/bin:$PATH" &&\
    conda config --set auto_update_conda false &&\
    conda install conda-build=3.0.6 &&\
    curl -LO https://storage.googleapis.com/golang/go1.7.linux-amd64.tar.gz &&\
    tar -C /usr/local -xzf go1.7.linux-amd64.tar.gz

ADD recipe-script.sh /root/

CMD chmod 755 recipe-script.sh && ./recipe-script.sh
