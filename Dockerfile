FROM golang:1.20
WORKDIR /app
COPY . ./
RUN git config --global --add safe.directory /app
RUN go build -v -o app
ENTRYPOINT ["/app/app"]
