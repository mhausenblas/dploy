FROM golang:1.6-wheezy
MAINTAINER Michael Hausenblas "michael.hausenblas@gmail.com"
ENV REFRESHED_AT 2016-04-30T20:19Z

RUN    go get github.com/mhausenblas/dploy \
    && go install github.com/mhausenblas/dploy

ENTRYPOINT ["/go/bin/dploy"]