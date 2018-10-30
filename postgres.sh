#!/bin/bash -e
if [ -z "${DBDIR}" ]; then
    DBDIR=`mktemp -d`
fi

PIDFILE=$DBDIR/.s.PGSQL.5432.lock

PG_VER=`pg_config --version | cut -d' ' -f 2`
if [[ $PG_VER == 10* ]]; then
    PG_VER="10"
elif [[ $PG_VER == 9* ]]; then
    PG_VER=`echo ${PG_VER} | cut -d'.' -f1,2`
fi

start () {
    /usr/lib/postgresql/${PG_VER}/bin/initdb -D $DBDIR
    cat >/$DBDIR/postgresql.conf <<EOF
    fsync = off
    listen_addresses = ''

    unix_socket_directories = '$DBDIR'
EOF
    /usr/lib/postgresql/${PG_VER}/bin/postgres -D $DBDIR &> $DBDIR/stdout.log &
}

stop() {
    if [ -e "$PIDFILE" ]; then
        PID=`head -1 $PIDFILE`
        if [ -n "$PID" ]; then
            kill -9 $PID
        fi
    fi
    rm -rf $DBDIR
}

trap "stop" SIGINT SIGTERM EXIT
start
PGDATABASE=postgres PGHOST="${DBDIR}" PGUSERNAME=postgres $@
