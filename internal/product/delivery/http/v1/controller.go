package v1

import (
	"github.com/AleksK1NG/go-elasticsearch/config"
	"github.com/AleksK1NG/go-elasticsearch/internal/dto"
	"github.com/AleksK1NG/go-elasticsearch/internal/metrics"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/constants"
	httpErrors "github.com/AleksK1NG/go-elasticsearch/pkg/http_errors"
	"github.com/AleksK1NG/go-elasticsearch/pkg/logger"
	"github.com/AleksK1NG/go-elasticsearch/pkg/tracing"
	"github.com/AleksK1NG/go-elasticsearch/pkg/utils"
	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go/log"
	uuid "github.com/satori/go.uuid"
	"net/http"
)

type productController struct {
	log            logger.Logger
	cfg            *config.Config
	productUseCase domain.ProductUseCase
	group          *echo.Group
	validate       *validator.Validate
	metrics        *metrics.SearchMicroserviceMetrics
}

func NewProductController(
	log logger.Logger,
	cfg *config.Config,
	productUseCase domain.ProductUseCase,
	group *echo.Group,
	validate *validator.Validate,
	metrics *metrics.SearchMicroserviceMetrics,
) *productController {
	return &productController{log: log, cfg: cfg, productUseCase: productUseCase, group: group, validate: validate, metrics: metrics}
}

func (h *productController) indexAsync() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, span := tracing.StartHttpServerTracerSpan(c, "productController.indexAsync")
		defer span.Finish()

		var product domain.Product
		if err := c.Bind(&product); err != nil {
			h.log.Errorf("(Bind) err: %v", tracing.TraceWithErr(span, err))
			return httpErrors.ErrorCtxResponse(c, err, h.cfg.Http.DebugErrorsResponse)
		}
		product.ID = uuid.NewV4().String()

		if err := h.productUseCase.IndexAsync(ctx, product); err != nil {
			h.log.Errorf("(productUseCase.IndexAsync) err: %v", tracing.TraceWithErr(span, err))
			return httpErrors.ErrorCtxResponse(c, err, h.cfg.Http.DebugErrorsResponse)
		}

		h.log.Infof("created product: %+v", product)
		h.metrics.HttpSuccessIndexAsyncRequests.Inc()
		return c.JSON(http.StatusCreated, product)
	}
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
		h.metrics.HttpSuccessIndexRequests.Inc()
		return c.JSON(http.StatusCreated, product)
	}
}

func (h *productController) search() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, span := tracing.StartHttpServerTracerSpan(c, "productController.search")
		defer span.Finish()

		searchTerm := c.QueryParam("search")
		pagination := utils.NewPaginationFromQueryParams(c.QueryParam(constants.Size), c.QueryParam(constants.Page))

		searchResult, err := h.productUseCase.Search(ctx, searchTerm, pagination)
		if err != nil {
			h.log.Errorf("(productUseCase.Search) err: %v", tracing.TraceWithErr(span, err))
			return httpErrors.ErrorCtxResponse(c, err, h.cfg.Http.DebugErrorsResponse)
		}

		h.log.Infof("search result: %s", searchResult.PaginationResponse.String())
		h.metrics.HttpSuccessSearchRequests.Inc()
		span.LogFields(log.String("search result", searchResult.PaginationResponse.String()))
		return c.JSON(http.StatusOK, dto.SearchProductsResponse{
			SearchTerm: searchTerm,
			Pagination: searchResult.PaginationResponse,
			Products:   searchResult.List,
		})
	}
}
