FROM golang:latest 
RUN mkdir /app 
ADD . /app/ 
WORKDIR /app 
RUN go get github.com/pkochubey/golang-test-task
RUN go build -o main . 
CMD ["/app/main"]