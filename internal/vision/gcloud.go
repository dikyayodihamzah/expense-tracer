package vision

import (
	"context"
	"fmt"

	visionapi "cloud.google.com/go/vision/v2/apiv1"
	"cloud.google.com/go/vision/v2/apiv1/visionpb"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

type gcloudProvider struct {
	client *visionapi.ImageAnnotatorClient
}

func newGCloud(ts oauth2.TokenSource) (*gcloudProvider, error) {
	if ts == nil {
		return nil, fmt.Errorf("google OAuth2 token source is required for gcloud vision provider")
	}
	ctx := context.Background()
	client, err := visionapi.NewImageAnnotatorClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("gcloud vision client: %w", err)
	}
	return &gcloudProvider{client: client}, nil
}

func (g *gcloudProvider) ExtractExpense(ctx context.Context, imageData []byte) (*model.Expense, error) {
	req := &visionpb.BatchAnnotateImagesRequest{
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image: &visionpb.Image{Content: imageData},
				Features: []*visionpb.Feature{
					{Type: visionpb.Feature_DOCUMENT_TEXT_DETECTION},
				},
			},
		},
	}

	resp, err := g.client.BatchAnnotateImages(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gcloud vision API call: %w", err)
	}

	if len(resp.GetResponses()) == 0 {
		return nil, fmt.Errorf("gcloud vision returned empty responses")
	}

	annotation := resp.GetResponses()[0].GetFullTextAnnotation()
	if annotation == nil {
		return nil, fmt.Errorf("gcloud vision returned no text annotation")
	}

	return &model.Expense{
		Description: annotation.GetText(),
		Category:    model.CategoryOthers,
	}, nil
}
