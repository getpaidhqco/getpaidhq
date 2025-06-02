# Payloop (wip)

Payloop is a payment processing system that allows users to create and manage subscriptions.
It is designed to be flexible and extensible, allowing developers to easily integrate it into their applications.

## Project Structure

Ref: https://github.com/sklinkert/go-ddd.git
This project uses the Domain Driven Design (DDD) principles

## Installation

For the development environment, we use Docker to run the application. The application is built using Go and uses
Postgres as the database.
Use the docker compose file to run the application and the database.

### Prerequisites

- Docker
- Docker Compose
- Go 1.24
- Temporal client

### Post installation

Create the `subscriptions` namespace in Temporal

```
temporal operator  namespace create -n subscriptions
```

Run the seed script to create the initial data in the database

## Notes on Change Data Capture (CDC)

Payloop uses two datases, the operational db `payloop` and the reporting db `payloop_reporting`.
The two databases are kept in sync using a change data capture (CDC) process (still testing it out)

Currently (Apr 2025) the CDC library doesn't update the publication records for the logical replication, which means
we need to manually update the publication records in the `pg_publication` table every time there's an update to the
CDC Stream service. We need to remove the current publication and subscription so that the system can create
a new one when the server starts.

```sql
SELECT *
FROM pg_publication;
SELECT *
FROM pg_replication_slots;

DROP PUBLICATION cdc_pub;
SELECT pg_terminate_backend(22081);
SELECT pg_drop_replication_slot('cdc_slot2');

```

## Authentication & Authorization

Payloop uses an authentication wrapper to auth incoming api requests.
At the moment, we support API Key, Cognito and Clerk authentication.
To enable or disable an authentication method, add or remove the `group:"authenticators"` FX tag from the
injection, or remove the FX DI in modules.go.



## Database Migrations

For the Postgres database we use Prisma to manage the database schema and migrations. Migrations in Test and Prod
environments
are managed by the CI/CD pipeline. Migrations are executed before the Payloop backend is built and deployed.
Check the buildspec.yml file for more details.

## Connecting to Test environment

Via the bastion

Payloop API port 8888->8081

```
ssh -o StrictHostKeyChecking=no -N -L  8888:payloop.temporal.temporal:8081 ec2-user@ec2-34-244-193-216.eu-west-1.compute.amazonaws.com -i cj-bastion-test.pem -v
```

Temporal UI 9999->8080

```
ssh -o StrictHostKeyChecking=no -N -L  9999:temporal-svc.temporal:8080 ec2-user@ec2-34-244-193-216.eu-west-1.compute.amazonaws.com -i cj-bastion-test.pem -v
```

## Create a ECR registry

```
docker pull golang:1.24-alpine

aws ecr create-repository --repository-name golang-1_24-alpine --profile=cj-test
aws ecr get-login-password --region eu-west-1 --profile=cj-test |  docker login --username AWS --password-stdin 329237115630.dkr.ecr.eu-west-1.amazonaws.com
docker tag golang:1.24-alpine 329237115630.dkr.ecr.eu-west-1.amazonaws.com/golang-1_24-alpine
docker push 329237115630.dkr.ecr.eu-west-1.amazonaws.com/golang-1_24-alpine
```

```
docker pull temporalio/auto-setup

aws ecr create-repository --repository-name temporalio_auto_setup --profile=cj-test
aws ecr get-login-password --region eu-west-1 --profile=cj-test |  docker login --username AWS --password-stdin 329237115630.dkr.ecr.eu-west-1.amazonaws.com
docker tag temporalio/auto-setup 329237115630.dkr.ecr.eu-west-1.amazonaws.com/temporalio_auto_setup
docker push 329237115630.dkr.ecr.eu-west-1.amazonaws.com/temporalio_auto_setup

docker pull temporalio/admin-tools

aws ecr create-repository --repository-name temporalio_admin_tools --profile=cj-test
aws ecr get-login-password --region eu-west-1 --profile=cj-test |  docker login --username AWS --password-stdin 329237115630.dkr.ecr.eu-west-1.amazonaws.com
docker tag temporalio/admin-tools 329237115630.dkr.ecr.eu-west-1.amazonaws.com/temporalio_admin_tools
docker push 329237115630.dkr.ecr.eu-west-1.amazonaws.com/temporalio_admin_tools

docker pull temporalio/ui
aws ecr create-repository --repository-name temporalio_ui --profile=cj-test
aws ecr get-login-password --region eu-west-1 --profile=cj-test |  docker login --username AWS --password-stdin 329237115630.dkr.ecr.eu-west-1.amazonaws.com
docker tag temporalio/ui 329237115630.dkr.ecr.eu-west-1.amazonaws.com/temporalio_ui
docker push 329237115630.dkr.ecr.eu-west-1.amazonaws.com/temporalio_ui


```