package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/mdwt/payloop-cart"
	"net/http"
	"payloop/internal/domain/carts"
	"payloop/internal/domain/sessions"
	"payloop/internal/lib"
	"payloop/internal/services"
)

type SessionController struct {
	sessionService services.SessionService
	cartService    services.CartService
	logger         lib.Logger
}

func NewSessionController(sessionService services.SessionService, cartService services.CartService, logger lib.Logger) SessionController {
	return SessionController{
		sessionService: sessionService,
		cartService:    cartService,
		logger:         logger,
	}
}

func (s SessionController) Create(c *gin.Context) {
	trxHandle := c.MustGet(lib.DBTransaction)
	var input sessions.CreateSessionRequest
	sessionId := lib.GenerateId("session")

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	s.logger.Debug("Creating session", "input", input)

	cartData := cart.New(cart.CreateCartOptions{
		Currency: input.Currency,
		Items:    make([]cart.Item, 0),
	})

	cartInstance, err := s.cartService.WithTrx(trxHandle).CreateCart(c.Request.Context(), carts.CreateCartInput{
		AccountId: input.AccountId,
		Cart:      cartData,
		Metadata:  nil,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	_, err = s.sessionService.WithTrx(trxHandle).CreateSession(
		c.Request.Context(),
		sessions.CreateSessionInput{
			AccountId: input.AccountId,
			CartId:    cartInstance.Id,
			Metadata:  nil,
		})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, sessions.CreateSessionResponse{
		Id:     sessionId,
		CartId: cartInstance.Id,
	})
}
