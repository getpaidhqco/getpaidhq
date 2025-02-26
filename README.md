

## Project Structure
Ref: https://github.com/sklinkert/go-ddd.git
This project uses the Domain Driven Design (DDD) principles 


## Project Structure
Ref: https://github.com/sklinkert/go-ddd.git
This project uses the Domain Driven Design (DDD) principles

Based on the provided Prisma schema, here are the proposed aggregates for the model:

## Aggregates

Determining the boundaries of an aggregate in your domain model involves understanding the business rules and invariants that need to be maintained consistently. Here are some guidelines to help you determine the boundaries:

1. **Single Responsibility**: Each aggregate should have a single responsibility and represent a cohesive unit of behavior.

2. **Consistency**: Identify the entities that need to be consistent together. These entities should be part of the same aggregate.

3. **Transactional Boundaries**: Aggregates define transactional boundaries. Changes to an aggregate should be committed as a single transaction.

4. **Access Control**: The aggregate root controls access to the other entities within the aggregate. Only the aggregate root should be directly accessed by other parts of the system.

5. **Business Rules**: Consider the business rules and invariants that need to be enforced. Entities that share these rules should be part of the same aggregate.

6. **Lifecycle**: Entities that have a shared lifecycle should be part of the same aggregate. For example, if deleting an order should also delete its order items, they should be part of the same aggregate.

7. **Size and Performance**: Aggregates should not be too large. Large aggregates can lead to performance issues. Aim for a balance between consistency and performance.

By following these guidelines, you can determine the appropriate boundaries for your aggregates in your domain model.


### Org Aggregate:
- Org
- UserOrg
- Setting

### User Aggregate:
- User
- UserOrg

### Product Aggregate:
- Product
- Variant
- Price

### Cart Aggregate:
- Cart
- Order

### Order Aggregate:
- Order
- OrderItem
- Payment
- Subscription

### Customer Aggregate:
- Customer
- PaymentMethod

### Subscription Aggregate:
- Subscription
- Payment

Each aggregate is designed to encapsulate related entities that need to be consistent together and share a common lifecycle.




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