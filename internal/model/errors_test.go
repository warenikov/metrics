package models

import "testing"

func TestAppError_Error(t *testing.T) {
	err := &AppError{Message: "test error", StatusCode: 400}
	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got %q", err.Error())
	}
}
