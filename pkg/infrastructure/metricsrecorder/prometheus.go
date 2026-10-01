package metricsrecorder

import (
	"errors"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

const (
	labelName    = "name"
	labelVersion = "version"

	stateInitialized = "initialized"
	stateAvailable   = "available"
	stateServed      = "served"
)

type Prometheus struct {
	archiveState      *prometheus.GaugeVec
	integrityFailures *prometheus.CounterVec
	uploadRequests    *prometheus.CounterVec
}

func NewPrometheus(registerer prometheus.Registerer) *Prometheus {
	return &Prometheus{
		archiveState: register(registerer, prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "registry_nsa_state",
			Help: "State of a NodeSolutionArchive on this node (1 when the state is reached).",
		}, []string{labelName, labelVersion, "state"})),
		integrityFailures: register(registerer, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "registry_nsa_integrity_failures_total",
			Help: "Number of integrity check failures of a solution archive, by stage.",
		}, []string{labelName, labelVersion, "stage"})),
		uploadRequests: register(registerer, prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "registry_nsa_upload_requests_total",
			Help: "Number of upload requests handled by the external API, by solution archive and HTTP status code.",
		}, []string{labelName, labelVersion, "code"})),
	}
}

// register reuses an already registered collector, so that several DI containers
// can live in the same process (e.g. integration tests) without panicking.
func register[C prometheus.Collector](registerer prometheus.Registerer, collector C) C {
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			if existing, ok := alreadyRegistered.ExistingCollector.(C); ok {
				return existing
			}
		}
		panic(err)
	}
	return collector
}

func (p *Prometheus) SetArchiveState(
	solutionArchive *domain.SolutionArchive,
	initialized, available, served bool,
) {
	for state, reached := range map[string]bool{
		stateInitialized: initialized,
		stateAvailable:   available,
		stateServed:      served,
	} {
		p.archiveState.WithLabelValues(solutionArchive.Name, solutionArchive.Version, state).Set(boolToFloat(reached))
	}
}

func (p *Prometheus) ForgetArchive(solutionArchive *domain.SolutionArchive) {
	labels := prometheus.Labels{labelName: solutionArchive.Name, labelVersion: solutionArchive.Version}
	p.archiveState.DeletePartialMatch(labels)
	p.integrityFailures.DeletePartialMatch(labels)
	p.uploadRequests.DeletePartialMatch(labels)
}

func (p *Prometheus) IncIntegrityFailure(solutionArchive *domain.SolutionArchive, stage domain.IntegrityStage) {
	p.integrityFailures.WithLabelValues(solutionArchive.Name, solutionArchive.Version, string(stage)).Inc()
}

func (p *Prometheus) IncUploadRequest(solutionArchive *domain.SolutionArchive, statusCode int) {
	p.uploadRequests.WithLabelValues(solutionArchive.Name, solutionArchive.Version, strconv.Itoa(statusCode)).Inc()
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
