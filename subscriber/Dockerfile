FROM golang:1.22.4

WORKDIR /work
COPY ./subscriber /work/subscriber
COPY ./lib /work/lib
RUN go install github.com/air-verse/air@v1.52.3

ENTRYPOINT [ "/work/subscriber/entrypoint.sh" ]
