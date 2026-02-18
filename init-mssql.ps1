# 1. Start Containers
docker-compose up -d

# 2. Wait for MSSQL to be ready (takes ~30s usually)
Start-Sleep -Seconds 30

# 3. Create Database & Run Schema Script (using sqlcmd inside container)
# Note: /opt/mssql-tools/bin/sqlcmd is the default path in the image
docker exec -i pump-mssql /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "StrongPass1!" -C -Q "CREATE DATABASE sakila;"
docker exec -i pump-mssql /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "StrongPass1!" -C -d sakila -i /var/opt/mssql/init.sql

Write-Host "MSSQL Initialization Complete!"
