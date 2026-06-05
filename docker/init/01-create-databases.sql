-- Bootstrap script for the shared Postgres instance.
-- POSTGRES_DB (getpaidhq) is created by the postgres image itself; this
-- script adds the auxiliary databases: reporting, the usage-event store, and Hatchet.

CREATE DATABASE getpaidhq_reports;
CREATE DATABASE getpaidhq_usage;
CREATE DATABASE hatchet;
