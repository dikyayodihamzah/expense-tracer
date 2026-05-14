package repository

import (
	"context"
	"fmt"

	vision "cloud.google.com/go/vision/v2/apiv1"
	"cloud.google.com/go/vision/v2/apiv1/visionpb"
	"google.golang.org/api/option"

	"github.com/dikyayodihamzah/expense-tracer/internal/model"
)

// gcloudVision implements VisionProvider using Google Cloud Vision API.
type gcloudVision struct {
	credentialsPath string
}

// newGCloudVision creates a new Google Cloud Vision provider.
func newGCloudVision(credentialsPath string) (VisionProvider, error) {
	if credentialsPath == "" {
		return nil, fmt.Errorf("credentials path is required for Google Cloud vision provider")
	}
	return &gcloudVision{credentialsPath: credentialsPath}, nil
}

// ExtractExpense sends the image to Google Cloud Vision and returns an Expense
// with the detected OCR text as the Description.
func (gv *gcloudVision) ExtractExpense(ctx context.Context, imageData []byte) (*model.Expense, error) {
	client, err := vision.NewImageAnnotatorClient(ctx, option.WithCredentialsFile(gv.credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("create gcloud vision client: %w", err)
	}
	defer client.Close()

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

	resp, err := client.BatchAnnotateImages(ctx, req)
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
