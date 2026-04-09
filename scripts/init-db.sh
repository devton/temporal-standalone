#!/bin/bash
set -e

# Cria os bancos de dados necessários para o Temporal e Casdoor
# Todos no mesmo PostgreSQL 18

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    -- Database para visibility (search/queries)
    CREATE DATABASE temporal_visibility;
    GRANT ALL PRIVILEGES ON DATABASE temporal_visibility TO temporal;
    
    -- Database para Casdoor (OIDC)
    CREATE USER casdoor WITH PASSWORD 'casdoor_secret';
    CREATE DATABASE casdoor OWNER casdoor;
    GRANT ALL PRIVILEGES ON DATABASE casdoor TO casdoor;
EOSQL

echo "Databases created successfully!"
