FROM golang:1.19
WORKDIR /app
COPY . ./
RUN git config --global --add safe.directory /app
RUN go build -v -o app
ENTRYPOINT ["/app/app"]
