package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"electronic-digital-signature/internal/domain/model"
)

type LocalInboundRepository struct {
	path string
	mu   sync.RWMutex
}

func NewLocalInboundRepository(basePath string) *LocalInboundRepository {
	return &LocalInboundRepository{
		path: filepath.Join(basePath, "inbound_packages.json"),
	}
}

func (r *LocalInboundRepository) Save(ctx context.Context, pkg model.InboundPackage) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	packages, err := r.readAllLocked()
	if err != nil {
		return err
	}

	replaced := false
	for index := range packages {
		if packages[index].MailMessageID == pkg.MailMessageID {
			packages[index] = pkg
			replaced = true
			break
		}
	}
	if !replaced {
		packages = append(packages, pkg)
	}

	sortInboundPackages(packages)
	return r.writeAllLocked(packages)
}

func (r *LocalInboundRepository) GetByMailMessageID(ctx context.Context, mailMessageID string) (*model.InboundPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	packages, err := r.readAllLocked()
	if err != nil {
		return nil, err
	}

	for index := range packages {
		if packages[index].MailMessageID == mailMessageID {
			copy := packages[index]
			return &copy, nil
		}
	}

	return nil, nil
}

func (r *LocalInboundRepository) List(ctx context.Context) ([]model.InboundPackage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	packages, err := r.readAllLocked()
	if err != nil {
		return nil, err
	}

	sortInboundPackages(packages)
	return append([]model.InboundPackage(nil), packages...), nil
}

func (r *LocalInboundRepository) readAllLocked() ([]model.InboundPackage, error) {
	if r.path == "" {
		return nil, fmt.Errorf("inbound repository path is not configured")
	}

	content, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.InboundPackage{}, nil
		}
		return nil, fmt.Errorf("read inbound repository: %w", err)
	}
	if len(content) == 0 {
		return []model.InboundPackage{}, nil
	}

	var packages []model.InboundPackage
	if err := json.Unmarshal(content, &packages); err != nil {
		return nil, fmt.Errorf("decode inbound repository: %w", err)
	}

	return packages, nil
}

func (r *LocalInboundRepository) writeAllLocked(packages []model.InboundPackage) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return fmt.Errorf("create inbound repository directory: %w", err)
	}

	content, err := json.MarshalIndent(packages, "", "  ")
	if err != nil {
		return fmt.Errorf("encode inbound repository: %w", err)
	}

	if err := os.WriteFile(r.path, content, 0o600); err != nil {
		return fmt.Errorf("write inbound repository: %w", err)
	}

	return nil
}

func sortInboundPackages(packages []model.InboundPackage) {
	sort.SliceStable(packages, func(i, j int) bool {
		left := packages[i].ProcessedAt
		if left.IsZero() {
			left = packages[i].MailReceivedAt
		}
		right := packages[j].ProcessedAt
		if right.IsZero() {
			right = packages[j].MailReceivedAt
		}

		return left.After(right)
	})
}
