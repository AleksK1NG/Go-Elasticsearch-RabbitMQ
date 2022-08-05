package domain

import (
	"github.com/AleksK1NG/go-elasticsearch/pkg/utils"
)

type ProductSearchResponse struct {
	List               []*Product                `json:"list"`
	PaginationResponse *utils.PaginationResponse `json:"paginationResponse"`
}
