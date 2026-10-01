package metricsrecorder

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

var testArchive = &domain.SolutionArchive{Name: "metalk8s", Version: "1.25.3"}

func TestSetArchiveState(t *testing.T) {
	p := NewPrometheus(prometheus.NewRegistry())

	p.SetArchiveState(testArchive, true, false, false)
	p.SetArchiveState(testArchive, true, true, false)

	for state, want := range map[string]float64{
		stateInitialized: 1,
		stateAvailable:   1,
		stateServed:      0,
	} {
		got := testutil.ToFloat64(p.archiveState.WithLabelValues("metalk8s", "1.25.3", state))
		if got != want {
			t.Errorf("state %q: got %v, want %v", state, got, want)
		}
	}
}

func TestForgetArchive(t *testing.T) {
	p := NewPrometheus(prometheus.NewRegistry())
	other := &domain.SolutionArchive{Name: "other", Version: "2.0.0"}

	p.SetArchiveState(testArchive, true, true, true)
	p.SetArchiveState(other, true, false, false)
	p.IncIntegrityFailure(testArchive, domain.IntegrityStageArchiveChecksum)
	p.IncUploadRequest(testArchive, 200)
	p.IncUploadRequest(other, 200)

	p.ForgetArchive(testArchive)

	if got := testutil.CollectAndCount(p.archiveState); got != 3 {
		t.Errorf("archive state series: got %d, want 3", got)
	}
	if got := testutil.CollectAndCount(p.integrityFailures); got != 0 {
		t.Errorf("integrity failure series: got %d, want 0", got)
	}
	if got := testutil.CollectAndCount(p.uploadRequests); got != 1 {
		t.Errorf("upload request series: got %d, want 1", got)
	}
}

func TestIncIntegrityFailure(t *testing.T) {
	p := NewPrometheus(prometheus.NewRegistry())

	p.IncIntegrityFailure(testArchive, domain.IntegrityStageChunkDigest)
	p.IncIntegrityFailure(testArchive, domain.IntegrityStageChunkDigest)
	p.IncIntegrityFailure(testArchive, domain.IntegrityStageArchiveChecksum)

	if got := testutil.ToFloat64(p.integrityFailures.WithLabelValues("metalk8s", "1.25.3", "chunk_digest")); got != 2 {
		t.Errorf("chunk_digest: got %v, want 2", got)
	}
	if got := testutil.ToFloat64(p.integrityFailures.WithLabelValues("metalk8s", "1.25.3", "archive_checksum")); got != 1 {
		t.Errorf("archive_checksum: got %v, want 1", got)
	}
}

func TestIncUploadRequest(t *testing.T) {
	p := NewPrometheus(prometheus.NewRegistry())

	p.IncUploadRequest(testArchive, 200)
	p.IncUploadRequest(testArchive, 400)
	p.IncUploadRequest(testArchive, 400)

	if got := testutil.ToFloat64(p.uploadRequests.WithLabelValues("metalk8s", "1.25.3", "400")); got != 2 {
		t.Errorf("code 400: got %v, want 2", got)
	}
}

func TestNewPrometheusReusesRegisteredCollectors(t *testing.T) {
	registry := prometheus.NewRegistry()
	first := NewPrometheus(registry)
	second := NewPrometheus(registry)

	first.IncUploadRequest(testArchive, 200)
	second.IncUploadRequest(testArchive, 200)

	if got := testutil.ToFloat64(first.uploadRequests.WithLabelValues("metalk8s", "1.25.3", "200")); got != 2 {
		t.Errorf("code 200: got %v, want 2", got)
	}
}
