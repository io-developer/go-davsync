FROM golang:1.15.2

ADD . /appsrc
RUN /appsrc/build.sh && mv /appsrc/bin/davsync /davsync && rm -rf /appsrc

ENTRYPOINT [ "/davsync" ]