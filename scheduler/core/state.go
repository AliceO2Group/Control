package core

import (
	"math/rand"
	"time"
	"sync"

	"github.com/mesos/mesos-go/api/v1/lib"
	"github.com/mesos/mesos-go/api/v1/lib/backoff"
	"github.com/mesos/mesos-go/api/v1/lib/scheduler/calls"
	"github.com/looplab/fsm"
	"gitlab.cern.ch/tmrnjava/test-scheduler/scheduler/core/environment"
	"github.com/pborman/uuid"
)

func newInternalState(cfg Config, shutdown func()) (*internalState, error) {
	metricsAPI := initMetrics(cfg)
	executorInfo, err := prepareExecutorInfo(
		cfg.executor,
		cfg.execImage,
		cfg.server,
		buildWantsExecutorResources(cfg),
		cfg.jobRestartDelay,
		metricsAPI,
	)
	if err != nil {
		return nil, err
	}
	creds, err := loadCredentials(cfg.credentials)
	if err != nil {
		return nil, err
	}
	state := &internalState{
		config:             cfg,
		reviveTokens:       backoff.BurstNotifier(cfg.reviveBurst, cfg.reviveWait, cfg.reviveWait, nil),
		resourceOffersDone: make(chan []uuid.Array),
		wantsTaskResources: mesos.Resources{},
		executor:           executorInfo,
		metricsAPI:         metricsAPI,
		cli:                buildHTTPSched(cfg, creds),
		random:             rand.New(rand.NewSource(time.Now().Unix())),
		shutdown:           shutdown,
		environments:		environment.NewEnvironments(),
	}
	return state, nil
}

type internalState struct {
	sync.RWMutex

	// needs locking:
	wantsTaskResources mesos.Resources
	tasksLaunched      int
	tasksFinished      int
	err                error

	// not used in multiple goroutines:
	executor           *mesos.ExecutorInfo
	reviveTokens       <-chan struct{}
	resourceOffersDone chan []uuid.Array
	random             *rand.Rand

	// shouldn't change at runtime, so thread safe:
	role               string
	cli                calls.Caller
	config             Config
	shutdown           func()

	// uses prometheus counters, so thread safe
	metricsAPI         *metricsAPI

	// uses locks, so thread safe
	sm                 *fsm.FSM
	environments	   *environment.Environments
}

