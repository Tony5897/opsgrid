-- Creates the application database owned by the migrator and locks down
-- connection rights. Run as a superuser after roles.sql. psql variable
-- :dbname selects the database name (default opsgrid).

SELECT format('CREATE DATABASE %I OWNER opsgrid_migrator', :'dbname')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'dbname')
\gexec

SELECT format('REVOKE ALL ON DATABASE %I FROM PUBLIC', :'dbname') \gexec
SELECT format(
    'GRANT CONNECT ON DATABASE %I TO opsgrid_app, opsgrid_worker, opsgrid_relay, opsgrid_reporting_ro',
    :'dbname'
) \gexec
