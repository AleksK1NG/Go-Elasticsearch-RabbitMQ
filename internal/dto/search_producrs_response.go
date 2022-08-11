package dto

import (
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/utils"
)

type SearchProductsResponse struct {
	SearchTerm string                    `json:"searchTerm"`
	Pagination *utils.PaginationResponse `json:"pagination"`
	Products   []*domain.Product         `json:"products"`
}
