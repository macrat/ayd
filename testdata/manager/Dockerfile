FROM golang:latest

WORKDIR /app

COPY . .
RUN go build -o manager .

EXPOSE 8080
CMD ["./manager"]
