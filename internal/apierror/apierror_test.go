package apierror

import (
	"encoding/json"
	"testing"
)

func TestResponseJSONRoundTrip(t *testing.T) {
	original := Response{
		Code:    "INVALID_REQUEST",
		Message: "field is required",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Response
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Code != original.Code {
		t.Errorf("Code: got %q, want %q", got.Code, original.Code)
	}
	if got.Message != original.Message {
		t.Errorf("Message: got %q, want %q", got.Message, original.Message)
	}
}

func TestResponse_JSONKeys(t *testing.T) {
	r := Response{Code: "NOT_FOUND", Message: "resource not found"}
	data, _ := json.Marshal(r)

	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	if _, ok := m["code"]; !ok {
		t.Error("expected JSON key 'code' to be present")
	}
	if _, ok := m["message"]; !ok {
		t.Error("expected JSON key 'message' to be present")
	}
}
