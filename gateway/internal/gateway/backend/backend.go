package backend

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	gw "github.com/cvmfs/gateway/internal/gateway"
	"github.com/cvmfs/gateway/internal/gateway/receiver"
	stats "github.com/cvmfs/gateway/internal/gateway/statistics"
)

// Services is a container for the various
// backend services
type Services struct {
	Config                gw.Config
	Access                AccessConfig
	DB                    *DB
	Pool                  *receiver.Pool
	Notifications         *NotificationSystem
	StatsMgr              *stats.StatisticsMgr
	LeaseNotificationChan chan string
}

// ActionController contains the various actions that can be performed with the backend
type ActionController interface {
	GetKey(ctx context.Context, keyID string) *KeyConfig
	GetRepo(ctx context.Context, repoName string) (*RepositoryConfig, error)
	GetRepos(ctx context.Context) (map[string]RepositoryConfig, error)
	SetRepoEnabled(ctx context.Context, repository string, enabled bool) error
	NewLease(ctx context.Context, keyID, leasePath, hostname string, protocolVersion int) (string, error)
	GetLeases(ctx context.Context) (map[string]LeaseDTO, error)
	GetLease(ctx context.Context, tokenStr string) (*LeaseDTO, error)
	CancelLeases(ctx context.Context, repoPath string) error
	CancelLease(ctx context.Context, tokenStr string) error
	CommitLease(ctx context.Context, tokenStr, oldRootHash, newRootHash string, tag gw.RepositoryTag) (uint64, error)
	SubmitPayload(ctx context.Context, token string, payload io.Reader, digest string, headerSize int) error
	RunGC(ctx context.Context, options GCOptions) (string, error)
	PublishManifest(ctx context.Context, repository string, message NotificationMessage)
	SubscribeToNotifications(ctx context.Context, repository string) SubscriberHandle
	UnsubscribeFromNotifications(ctx context.Context, repository string, handle SubscriberHandle) error
	AcceptRingToken(ctx context.Context, repository string) error
	PostRingToken(repository string) error
	AddToRing(ctx context.Context, tx *sql.Tx, repository string, hostName string) error
	RemoveLocally(ctx context.Context, repository string, hostName string) error
	HasRingToken(ctx context.Context, repository string) bool
	CanStartLease(ctx context.Context, repository string) bool
	GetRingGateways(ctx context.Context, tx *sql.Tx, repository string) ([]string, error)
	GetRingGatewaysStatus(ctx context.Context, repository string) ([]gwStatus, error)
	GetNextRingGateway(ctx context.Context, repository string, currentAddress string, maxStatus int) (string, error)
	RequestAddition(ctx context.Context, tx *sql.Tx, repository string, hostName string) error
	RequestRemoval(ctx context.Context, repository string, hostName string) error
	InvalidateToken(ctx context.Context, repository string)
	SendInvalidationRequest(ctx context.Context, repository string) error
	SetGwStatus(ctx context.Context, repository string, address string, status int) error
}

// GetKey returns the key configuration associated with a key ID
func (s *Services) GetKey(ctx context.Context, keyID string) *KeyConfig {
	return s.Access.GetKeyConfig(keyID)
}

// StartBackend initializes the various backend services
func StartBackend(cfg gw.Config) (*Services, error) {
	ac, err := NewAccessConfig(cfg.AccessConfigFile)
	if err != nil {
		return nil, fmt.Errorf("loading repository access configuration failed: %w", err)
	}

	db, err := OpenDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not create lease DB: %w", err)
	}

	smgr := stats.NewStatisticsMgr()

	pool, err := receiver.StartPool(cfg.ReceiverPath, cfg.NumReceivers, cfg.MockReceiver, smgr)
	if err != nil {
		return nil, fmt.Errorf("could not start receiver pool: %w", err)
	}

	ns, err := NewNotificationSystem(cfg.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("could not initialize notification system: %w", err)
	}

	services := Services{Config: cfg, Access: *ac, DB: db, Pool: pool, Notifications: ns, StatsMgr: smgr, LeaseNotificationChan: make(chan string)}

	if err := PopulateRepositories(&services); err != nil {
		return nil, fmt.Errorf("could not populate repository table: %w", err)
	}

	if cfg.EnableMultiGateway {
		if err := InitTokenRing(&services); err != nil {
			return nil, fmt.Errorf("could not initialize token ring: %w", err)
		}
	}

	return &services, nil
}

// Stop all the backend services
func (s *Services) Stop() error {
	if err := s.DB.Close(); err != nil {
		return fmt.Errorf("could not close database: %w", err)
	}
	return nil
}

func PopulateRepositories(s *Services) error {
	ctx := context.Background()
	if err := s.DeleteAllRepositories(ctx); err != nil {
		return err
	}
	for name := range s.Access.Repositories {
		if err := s.NewRepo(context.Background(), name, true); err != nil {
			return err
		}
	}

	return nil
}
