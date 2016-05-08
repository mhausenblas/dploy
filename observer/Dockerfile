FROM golang:1.6-wheezy
MAINTAINER Michael Hausenblas "michael.hausenblas@gmail.com"
ENV REFRESHED_AT 2016-05-08T15:54Z

RUN    go get github.com/mhausenblas/dploy/observer \
    && go install github.com/mhausenblas/dploy/observer

ENTRYPOINT ["/go/bin/observer"]