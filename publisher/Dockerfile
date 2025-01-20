FROM golang:1.22.4

WORKDIR /work
COPY ./publisher /work/publisher
COPY ./lib /work/lib
RUN go install github.com/air-verse/air@v1.52.3

ENTRYPOINT [ "/work/publisher/entrypoint.sh" ]
