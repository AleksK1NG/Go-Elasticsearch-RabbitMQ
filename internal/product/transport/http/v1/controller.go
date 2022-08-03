package v1

import (
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	httpErrors "github.com/AleksK1NG/go-elasticsearch/pkg/http_errors"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/AleksK1NG/go-elasticsearch/pkg/tracing"
	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	uuid "github.com/satori/go.uuid"
	"net/http"
)

type productController struct {
	log            logger.Logger
	cfg            *config.Config
	productUseCase domain.ProductUseCase
	group          *echo.Group
	validate       *validator.Validate
}

func NewProductController(
	log logger.Logger,
	cfg *config.Config,
	productUseCase domain.ProductUseCase,
	group *echo.Group,
	validate *validator.Validate,
) *productController {
	return &productController{log: log, cfg: cfg, productUseCase: productUseCase, group: group, validate: validate}
}

func (h *productController) index() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, span := tracing.StartHttpServerTracerSpan(c, "productController.index")
		defer span.Finish()

		var product domain.Product
		if err := c.Bind(&product); err != nil {
			h.log.Errorf("(Bind) err: %v", tracing.TraceWithErr(span, err))
			return httpErrors.ErrorCtxResponse(c, err, h.cfg.Http.DebugErrorsResponse)
		}
		product.ID = uuid.NewV4().String()

		if err := h.productUseCase.Index(ctx, product); err != nil {
			h.log.Errorf("(productUseCase.Index) err: %v", tracing.TraceWithErr(span, err))
			return httpErrors.ErrorCtxResponse(c, err, h.cfg.Http.DebugErrorsResponse)
		}

		h.log.Infof("created product: %+v", product)
		return c.JSON(http.StatusCreated, product)
	}
}

func (h *productController) search() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, span := tracing.StartHttpServerTracerSpan(c, "productController.search")
		defer span.Finish()

		searchResult, err := h.productUseCase.Search(ctx, c.QueryParam("search"))
		if err != nil {
			h.log.Errorf("(productUseCase.Search) err: %v", tracing.TraceWithErr(span, err))
			return httpErrors.ErrorCtxResponse(c, err, h.cfg.Http.DebugErrorsResponse)
		}

		h.log.Infof("created product: %+v", searchResult)
		return c.JSON(http.StatusCreated, searchResult)
	}
}
