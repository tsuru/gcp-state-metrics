FROM golang:1.16

WORKDIR /app/

COPY . .

RUN go mod download

RUN go build -o app

ENV PORT=8080

EXPOSE 8080

CMD ["app"]