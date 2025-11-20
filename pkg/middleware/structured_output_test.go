package middleware

import (
	"context"
	"testing"

	"github.com/astercloud/aster/pkg/structured"
	"github.com/astercloud/aster/pkg/types"
)

func TestStructuredOutputMiddleware_Success(t *testing.T) {
	mw, err := NewStructuredOutputMiddleware(&StructuredOutputMiddlewareConfig{
		Spec: structured.OutputSpec{
			Enabled:        true,
			RequiredFields: []string{"foo"},
		},
	})
	if err != nil {
		t.Fatalf("create middleware: %v", err)
	}

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{Content: `{"foo":"bar","num":1}`},
		}, nil
	}

	resp, err := mw.WrapModelCall(context.Background(), &ModelRequest{}, handler)
	if err != nil {
		t.Fatalf("wrap call: %v", err)
	}

	if resp.Metadata == nil {
		t.Fatalf("expected metadata to be set")
	}

	if resp.Metadata["structured_error"] != nil {
		t.Fatalf("unexpected parse error: %v", resp.Metadata["structured_error"])
	}

	if resp.Metadata["structured_data"] == nil {
		t.Fatalf("structured_data not populated")
	}

	if missing, ok := resp.Metadata["structured_missing_fields"].([]string); ok && len(missing) != 0 {
		t.Fatalf("unexpected missing fields: %v", missing)
	}
}

func TestStructuredOutputMiddleware_MissingRequired(t *testing.T) {
	mw, err := NewStructuredOutputMiddleware(&StructuredOutputMiddlewareConfig{
		Spec: structured.OutputSpec{
			Enabled:        true,
			RequiredFields: []string{"foo", "bar"},
		},
	})
	if err != nil {
		t.Fatalf("create middleware: %v", err)
	}

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{Content: `{"foo":"bar"}`},
		}, nil
	}

	resp, err := mw.WrapModelCall(context.Background(), &ModelRequest{}, handler)
	if err != nil {
		t.Fatalf("wrap call: %v", err)
	}

	if resp.Metadata == nil {
		t.Fatalf("expected metadata to be set")
	}

	missing, ok := resp.Metadata["structured_missing_fields"].([]string)
	if !ok || len(missing) != 1 || missing[0] != "bar" {
		t.Fatalf("missing fields not recorded, got: %#v", resp.Metadata["structured_missing_fields"])
	}
}

func TestStructuredOutputMiddleware_ErrorWhenDisallow(t *testing.T) {
	mw, err := NewStructuredOutputMiddleware(&StructuredOutputMiddlewareConfig{
		Spec: structured.OutputSpec{
			Enabled: true,
		},
		AllowError: false,
	})
	if err != nil {
		t.Fatalf("create middleware: %v", err)
	}

	handler := func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
		return &ModelResponse{
			Message: types.Message{Content: "no json here"},
		}, nil
	}

	_, err = mw.WrapModelCall(context.Background(), &ModelRequest{}, handler)
	if err == nil {
		t.Fatalf("expected error when parsing failed with AllowError=false")
	}
}
