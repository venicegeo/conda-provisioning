#!/bin/bash
yum -y install epel-release && yum -y update
yum -y install zlib-devel wget gcc gcc-c++ autogen libtool make autoconf automake freetype-devel SDL-devel bzip2 swig openssl-devel
wget https://people.centos.org/tru/devtools-2/6/x86_64/RPMS/devtoolset-2-gcc-gfortran-4.8.2-15.el6.x86_64.rpm
yum -y install devtoolset-2-gcc-gfortran-4.8.2-15.el6.x86_64.rpm


