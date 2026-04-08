#!/bin/sh
# Initialize Casdoor database

psql -v ON_ERROR_STOP=1 --username "temporal" --dbname "postgres" <<-EOSQL
    CREATE USER casdoor WITH PASSWORD 'casdoor_secret';
    CREATE DATABASE casdoor OWNER casdoor;
    GRANT ALL PRIVILEGES ON DATABASE casdoor TO casdoor;
EOSQL

echo "Casdoor database created successfully!"
