package railway

import (
	"net/http"

	"github.com/Khan/genqlient/graphql"
)

type AuthedTransport struct {
	token   string
	wrapped http.RoundTripper
}

type RailwayClient struct {
	graphql.Client
}

func (t *AuthedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Content-Type", "application/json")
	return t.wrapped.RoundTrip(req)
}

func NewAuthedClient(token string) *RailwayClient {
	httpClient := http.Client{
		Transport: &AuthedTransport{
			token:   token,
			wrapped: http.DefaultTransport,
		},
	}

	return &RailwayClient{
		graphql.NewClient("https://backboard.railway.app/graphql/internal", &httpClient),
	}
}
