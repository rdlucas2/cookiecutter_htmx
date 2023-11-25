#intended to be used for local development by mounting pwd as volume
FROM golang:1.19 as local
ENTRYPOINT ["/bin/bash"]

FROM golang:1.19 as base
WORKDIR /app
COPY go.mod go.mod
COPY go.sum go.sum
COPY ./src /app/src/

FROM base as test
COPY ./test.sh .
ENTRYPOINT ["./test.sh"]

FROM base as run
ENTRYPOINT [ "go", "run", "." ]

#TODO: compile to binary?
#FROM base as artifact
