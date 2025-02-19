package services

import (
	"context"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/lib/events"
	"payloop/internal/application/lib/events/topic"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type ProductService struct {
	productRepository repositories.ProductRepository
	cartRepository    repositories.CartRepository
	pubsub            events.PubSub
	logger            logger.Logger
}

func NewProductService(productRepository repositories.ProductRepository,
	cartRepository repositories.CartRepository,
	logger logger.Logger,
	pubsub events.PubSub,
) ProductService {
	return ProductService{
		productRepository: productRepository,
		cartRepository:    cartRepository,
		logger:            logger,
		pubsub:            pubsub,
	}
}

func (s ProductService) CreateProduct(ctx context.Context, input entities.CreateProductInput) (entities.Product, error) {

	product, err := s.productRepository.Create(ctx,
		entities.Product{
			OrgId: input.OrgId,
			Id:    lib.GenerateId("product"),
			Name:  input.Name,
		})

	_ = s.pubsub.Publish(input.OrgId, topic.ProductCreated, product)
	return product, err
}

func (s ProductService) List(ctx context.Context, orgId string, pagination request.Pagination) ([]entities.Product, int, error) {
	subs, total, err := s.productRepository.Find(ctx, orgId, pagination)
	if err != nil {
		s.logger.Error("Failed to list products", err.Error())
		return nil, 0, err
	}

	return subs, total, nil
}

func (s ProductService) FindById(ctx context.Context, orgId string, id string) (entities.Product, error) {
	product, err := s.productRepository.FindById(ctx, orgId, id)
	if err != nil {
		s.logger.Error("Failed to list products", err.Error())
		return entities.Product{}, err
	}

	return product, nil
}
