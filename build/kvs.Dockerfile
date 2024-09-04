FROM golang:1.23 as builder

LABEL authors="log"

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN make build

FROM scratch

COPY --from=builder /app/main .

CMD [ "./main" ]
