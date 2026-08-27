package runtimelease

import (
	"errors"
	"testing"
)

func TestProductAndCoreLeases(t *testing.T) {
	leases.products = [2]*Configuration{}
	t.Cleanup(func() { leases.products = [2]*Configuration{} })
	rtc := Configuration{AppID: "rtc", Endpoint: "rtc.example", CacheDir: "/cache", ConsoleLogEnabled: true}
	starts := 0
	if err := Init(RTC, rtc, func() error { starts++; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := Init(RTC, rtc, func() error { starts++; return nil }); err != nil {
		t.Fatal(err)
	}
	if starts != 1 {
		t.Fatalf("repeated lease started native %d times", starts)
	}
	cloudStorage := Configuration{AppID: "cloud-storage", Endpoint: "cloud-storage.example", CacheDir: "/cache", ConsoleLogEnabled: true}
	if err := Init(CloudStorage, cloudStorage, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	cloudStorage.CacheDir = "/other"
	if err := Init(CloudStorage, cloudStorage, func() error { t.Fatal("conflict reached native start"); return nil }); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflicting lease = %v", err)
	}
	if err := Shutdown(CloudStorage, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := Shutdown(RTC, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestFailedStartDoesNotCommitLease(t *testing.T) {
	leases.products = [2]*Configuration{}
	t.Cleanup(func() { leases.products = [2]*Configuration{} })
	want := errors.New("start failed")
	configuration := Configuration{CacheDir: "/cache"}
	if err := Init(RTC, configuration, func() error { return want }); !errors.Is(err, want) {
		t.Fatalf("failed start = %v", err)
	}
	if err := Init(RTC, configuration, func() error { return nil }); err != nil {
		t.Fatalf("retry after failed start: %v", err)
	}
}
