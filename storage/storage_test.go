package storage

import (
	"errors"
	"testing"
)

type fakeListReleaser struct {
	codes []int32
	calls int
}

func (r *fakeListReleaser) Close() int32 {
	r.calls++
	if len(r.codes) == 0 {
		return 0
	}
	code := r.codes[0]
	r.codes = r.codes[1:]
	return code
}

func TestCloudStorageRetainsFailedListReleaseForExplicitRetry(t *testing.T) {
	request := &fakeListReleaser{codes: []int32{0}}
	cloudStorage := &CloudStorage{listActive: true}
	cloudStorage.finishList(request, 6026)
	if cloudStorage.listActive || cloudStorage.retainedList == nil {
		t.Fatal("failed list release was not retained")
	}
	if err := cloudStorage.retryRetainedList(); err != nil {
		t.Fatalf("retry retained list: %v", err)
	}
	if request.calls != 1 || cloudStorage.retainedList != nil {
		t.Fatalf("retry calls=%d retained=%v", request.calls, cloudStorage.retainedList != nil)
	}
}

func TestCloudStorageKeepsRetainedListWhenRetryStillFails(t *testing.T) {
	request := &fakeListReleaser{codes: []int32{6026}}
	cloudStorage := &CloudStorage{retainedList: request}
	if err := cloudStorage.retryRetainedList(); !errors.Is(err, ErrInUse) {
		t.Fatalf("retry error=%v want ErrInUse", err)
	}
	if request.calls != 1 || cloudStorage.retainedList == nil {
		t.Fatalf("retry calls=%d retained=%v", request.calls, cloudStorage.retainedList != nil)
	}
}
