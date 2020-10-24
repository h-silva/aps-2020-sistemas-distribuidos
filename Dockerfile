FROM golang:latest

WORKDIR /usr/src/app

# Bundle app source
COPY . .

RUN go build -o tccunip-api

#ENV
ENV SERVER_PORT=3000
ENV PGHOST=maptree.cv1p4n0wnzfm.us-east-1.rds.amazonaws.com
ENV PGUSER=postgres
ENV PGDATABASE=maptree
ENV PGPASSWORD=heitor123
ENV PGPORT=5432

EXPOSE 3000

CMD [ "./tccunip-api" ]
