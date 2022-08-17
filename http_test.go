package http

import (
	"context"
	"testing"
)

func TestDownload(t *testing.T) {
	err := Download(context.Background(), "https://image.taikongsha.cc/18296213_1654078468_235510_2.amr", "E:\\test\\a.amr")
	if err != nil {
		t.Error(err)
	}
}
