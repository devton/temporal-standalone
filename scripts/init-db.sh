#!/bin/bash
set -e

# Cria os bancos de dados necessários para o Temporal
# Temporal requer múltiplos bancos para diferentes propósitos

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    -- Database para visibility
    CREATE DATABASE temporal_visibility;
    GRANT ALL PRIVILEGES ON DATABASE temporal_visibility TO temporal;
EOSQL

echo "Databases created successfully!"
