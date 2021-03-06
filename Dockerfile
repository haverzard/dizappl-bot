FROM golang:1.15.0-alpine

RUN apk update && apk add gcc g++ git make
RUN go get -u github.com/line/line-bot-sdk-go/linebot
RUN go get -u gorm.io/gorm
RUN go get -u gorm.io/driver/sqlite

RUN adduser -D -u 1001 -s /bin/bash haverzard

RUN mkdir /home/haverzard/dizappl

COPY . /home/haverzard/dizappl

WORKDIR /home/haverzard/dizappl
RUN go build -o main

EXPOSE 5000

CMD ["./main"]