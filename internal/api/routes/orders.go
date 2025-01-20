package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/lib"
)

type OrderRoutes struct {
	handler         lib.RequestHandler
	orderController controllers.OrderController
}
