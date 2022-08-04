package v1

import (
	"context"
	"encoding/json"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/http_client"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func TestIndexProduct(t *testing.T) {

	t.Parallel()

	client := http_client.NewHttpClient(true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	product := domain.Product{
		ID:           uuid.NewV4().String(),
		Title:        "Water",
		Description:  "some water",
		ImageURL:     "Image URL",
		CountInStock: 55555,
		Shop:         "Moscow shop",
		CreatedAt:    time.Now().UTC(),
	}

	t.Logf("product id: %s", product.ID)

	response, err := client.R().
		SetContext(ctx).
		SetBody(product).
		Post("http://localhost:8000/api/v1/products")
	require.NoError(t, err)
	require.NotNil(t, response)
	require.False(t, response.IsError())
	require.True(t, response.IsSuccess())
	require.Equal(t, response.StatusCode(), http.StatusCreated)

	var id string
	err = json.Unmarshal(response.Body(), &id)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	t.Logf("response: %s", response.String())
}