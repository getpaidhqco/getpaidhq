-- Bootstrap script for the shared Postgres instance.
-- POSTGRES_DB (getpaidhq) is created by the postgres image itself; this
-- script adds the auxiliary databases used by reporting and Hatchet.

CREATE DATABASE getpaidhq_reports;
CREATE DATABASE hatchet;
