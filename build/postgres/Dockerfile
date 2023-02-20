FROM postgres:15.1

RUN apt-get update && apt-get install -y git build-essential postgresql-server-dev-all

RUN git clone https://github.com/omniti-labs/pg_amqp.git && \
    cd pg_amqp && \
    make && \
    make install && \
    mkdir -p /usr/local/share/postgresql/ && \
    echo "shared_preload_libraries = 'pg_amqp'" >> /usr/local/share/postgresql/postgresql.conf.sample

CMD ["postgres"]