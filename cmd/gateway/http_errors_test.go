package main

import (
	"errors"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHTTPStatusFromGRPC(t *testing.T) {
	t.Run("InvalidArgument -> 400", func(t *testing.T) {
		err := status.Error(codes.InvalidArgument, "bad")
		gotStatus, gotCode, _ := httpStatusFromGRPC(err)
		if gotStatus != http.StatusBadRequest || gotCode != "INVALID_ARGUMENT" {
			t.Fatalf("got (%d,%s)", gotStatus, gotCode)
		}
	})

	t.Run("NotFound -> 404", func(t *testing.T) {
		err := status.Error(codes.NotFound, "missing")
		gotStatus, gotCode, _ := httpStatusFromGRPC(err)
		if gotStatus != http.StatusNotFound || gotCode != "NOT_FOUND" {
			t.Fatalf("got (%d,%s)", gotStatus, gotCode)
		}
	})

	t.Run("Unavailable -> 503", func(t *testing.T) {
		err := status.Error(codes.Unavailable, "down")
		gotStatus, gotCode, _ := httpStatusFromGRPC(err)
		if gotStatus != http.StatusServiceUnavailable || gotCode != "UNAVAILABLE" {
			t.Fatalf("got (%d,%s)", gotStatus, gotCode)
		}
	})

	t.Run("DeadlineExceeded -> 503", func(t *testing.T) {
		err := status.Error(codes.DeadlineExceeded, "timeout")
		gotStatus, gotCode, _ := httpStatusFromGRPC(err)
		if gotStatus != http.StatusServiceUnavailable || gotCode != "UNAVAILABLE" {
			t.Fatalf("got (%d,%s)", gotStatus, gotCode)
		}
	})

	t.Run("non-grpc error -> 500", func(t *testing.T) {
		err := errors.New("boom")
		gotStatus, gotCode, _ := httpStatusFromGRPC(err)
		if gotStatus != http.StatusInternalServerError || gotCode != "INTERNAL" {
			t.Fatalf("got (%d,%s)", gotStatus, gotCode)
		}
	})
}
