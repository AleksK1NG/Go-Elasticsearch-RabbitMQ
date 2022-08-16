package v1

import (
	"context"
	"github.com/AleksK1NG/go-elasticsearch/internal/product/domain"
	"github.com/AleksK1NG/go-elasticsearch/pkg/http_client"
	"github.com/brianvoe/gofakeit/v6"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"net/http"
	"testing"
	"time"
)

func TestIndexAsync(t *testing.T) {

	t.Parallel()

	client := http_client.NewHttpClient(true)

	ctx, cancel := context.WithTimeout(context.Background(), 165*time.Second)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < 10; i++ {
		g.Go(func() error {
			for j := 0; j < 500; j++ {
				product := domain.Product{
					ID:           uuid.NewV4().String(),
					Title:        gofakeit.Breakfast(),
					Description:  gofakeit.LoremIpsumSentence(60),
					ImageURL:     gofakeit.URL(),
					CountInStock: gofakeit.Int64(),
					Shop:         gofakeit.Company(),
					CreatedAt:    time.Now().UTC(),
				}

				t.Logf("product: %+v", product)

				response, err := client.R().
					SetContext(ctx).
					SetBody(product).
					Post("http://localhost:8000/api/v1/products/async")
				if err != nil {
					return err
				}
				t.Logf("response: %s", response.String())
			}
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err)
}

func TestIndexProduct(t *testing.T) {

	t.Parallel()

	client := http_client.NewHttpClient(true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	product := domain.Product{
		ID:           uuid.NewV4().String(),
		Title:        gofakeit.Breakfast(),
		Description:  gofakeit.LoremIpsumSentence(60),
		ImageURL:     gofakeit.URL(),
		CountInStock: gofakeit.Int64(),
		Shop:         gofakeit.Company(),
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
}
