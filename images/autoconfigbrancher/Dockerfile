FROM centos:7

ADD determinize-ci-operator /usr/bin/determinize-ci-operator
ADD config-brancher /usr/bin/config-brancher
ADD ci-operator-prowgen /usr/bin/ci-operator-prowgen
ADD autoconfigbrancher /usr/bin/autoconfigbrancher
ADD ci-op-configs-mirror /usr/bin/ci-op-configs-mirror

RUN yum install -y git && \
    yum clean all && \
    rm -rf /var/cache/yum

ENTRYPOINT ["/usr/bin/autoconfigbrancher"]
