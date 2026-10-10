package anonymousidentityservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/anonymousnames"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrLocked        = errors.New("anonymous.nameLocked")
	ErrQuota         = errors.New("anonymous.dailyLimit")
	ErrCandidate     = errors.New("anonymous.candidateExpired")
	ErrUnavailable   = errors.New("anonymous.unavailable")
	ErrInvalidParams = errors.New("common.request.invalidParams")
)

type PublicPersona struct {
	Kind       string `json:"kind"`
	PublicUID  string `json:"publicUid"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatarUrl"`
	ProfileURL string `json:"profileUrl"`
}

func Public(p identity.Persona) PublicPersona {
	return PublicPersona{"persona", p.UID, p.Name, "/a/" + p.UID + "/avatar.svg", "/a/" + p.UID}
}

type State struct {
	Persona               *PublicPersona   `json:"persona"`
	NameSelectedAt        *time.Time       `json:"nameSelectedAt"`
	NameChangeAvailableAt *time.Time       `json:"nameChangeAvailableAt"`
	Disabled              bool             `json:"disabled"`
	GovernanceDisabled    bool             `json:"governanceDisabled"`
	ShowContent           bool             `json:"showContent"`
	Day                   string           `json:"day"`
	Remaining             int              `json:"remaining"`
	ResetsAt              time.Time        `json:"resetsAt"`
	Batches               []identity.Batch `json:"batches"`
	LexiconVersion        string           `json:"lexiconVersion"`
}

type Service struct {
	DB   *gorm.DB
	Now  func() time.Time
	Draw func() ([]string, error)
}

func Default(ctx context.Context) Service {
	return Service{db.ConnectContext(ctx), time.Now, anonymousnames.Batch}
}
func randomID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}
func personaForOwner(tx *gorm.DB, owner uint64) (identity.Persona, error) {
	var binding identity.Binding
	var persona identity.Persona
	if err := tx.First(&binding, "owner_id = ?", owner).Error; err != nil {
		return persona, err
	}
	err := tx.First(&persona, "uid = ?", binding.PersonaUID).Error
	return persona, err
}
func (s Service) State(owner uint64) (State, error) {
	state := State{Remaining: 10, Batches: []identity.Batch{}, LexiconVersion: anonymousnames.Version, ShowContent: true}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		// Reads serialize with writes to return a consistent remaining/batch snapshot.
		if _, err := users.LockAnonymousOwnerForReadTx(tx, owner); err != nil {
			return err
		}
		day, next := anonymousnames.Day(s.Now())
		state.Day, state.ResetsAt = day, next
		p, err := personaForOwner(tx, owner)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			public := Public(p)
			state.Persona = &public
			state.NameSelectedAt = &p.NameSelectedAt
			state.NameChangeAvailableAt = &p.NameChangeAvailableAt
			state.Disabled = p.Disabled
			state.GovernanceDisabled = p.GovernanceDisabled
			state.ShowContent = p.ShowContent
		}
		var q identity.Quota
		if err := tx.First(&q, "owner_id = ? AND day = ?", owner, day).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		state.Remaining = 10 - q.Used
		return tx.Where("owner_id = ? AND day = ?", owner, day).Order("created_at, id").Find(&state.Batches).Error
	})
	return state, err
}
func (s Service) Generate(owner uint64, day, key string) (identity.Batch, error) {
	var result identity.Batch
	if len(key) < 8 || len(key) > 128 {
		return result, ErrInvalidParams
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := users.LockAnonymousOwnerTx(tx, owner); err != nil {
			return err
		}
		// Sample time after serialization; queued requests can cross Shanghai midnight.
		now := s.Now()
		current, next := anonymousnames.Day(now)
		if day != current {
			return ErrCandidate
		}
		err := tx.First(&result, "owner_id = ? AND day = ? AND request_key = ?", owner, day, key).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		p, err := personaForOwner(tx, owner)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			if p.Disabled || p.GovernanceDisabled {
				return ErrUnavailable
			}
			if now.Before(p.NameChangeAvailableAt) {
				return ErrLocked
			}
		}
		q := identity.Quota{OwnerID: owner, Day: day}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&q).Error; err != nil {
			return err
		}
		increment := tx.Model(&identity.Quota{}).Where("owner_id = ? AND day = ? AND used < 10", owner, day).UpdateColumn("used", gorm.Expr("used + 1"))
		if increment.Error != nil {
			return increment.Error
		}
		if increment.RowsAffected != 1 {
			return ErrQuota
		}
		words, err := s.Draw()
		if err != nil {
			return err
		}
		id, err := randomID()
		if err != nil {
			return err
		}
		result = identity.Batch{ID: id, OwnerID: owner, Day: day, RequestKey: key, Words: words, ExpiresAt: next, CreatedAt: now}
		return tx.Create(&result).Error
	})
	return result, err
}
func (s Service) Confirm(owner uint64, batchID string, index int) (PublicPersona, error) {
	var result PublicPersona
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := users.LockAnonymousOwnerTx(tx, owner); err != nil {
			return err
		}
		now := s.Now()
		day, _ := anonymousnames.Day(now)
		var batch identity.Batch
		if err := tx.First(&batch, "id = ? AND owner_id = ? AND day = ?", batchID, owner, day).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCandidate
			}
			return err
		}
		if !now.Before(batch.ExpiresAt) || index < 0 || index >= len(batch.Words) {
			return ErrCandidate
		}
		p, err := personaForOwner(tx, owner)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			var binding identity.Binding
			if err := tx.First(&binding, "owner_id = ?", owner).Error; err != nil {
				return err
			}
			if p.Disabled || p.GovernanceDisabled {
				return ErrUnavailable
			}
			// Response-loss and concurrent confirmation retries do not rename or extend the lock.
			if binding.SelectionBatch == batchID && binding.SelectionIndex == index {
				result = Public(p)
				return nil
			}
			if now.Before(p.NameChangeAvailableAt) {
				return ErrLocked
			}
			if p.Name == batch.Words[index] {
				result = Public(p)
				return nil
			}
			p.Name = batch.Words[index]
			p.NameSelectedAt = now
			p.NameChangeAvailableAt = anonymousnames.Anniversary(now)
			if err := tx.Save(&p).Error; err != nil {
				return err
			}
			if err := tx.Model(&binding).Updates(map[string]any{"selection_batch": batchID, "selection_index": index}).Error; err != nil {
				return err
			}
		} else {
			uid, err := randomID()
			if err != nil {
				return err
			}
			seed, err := randomID()
			if err != nil {
				return err
			}
			p = identity.Persona{UID: uid, Name: batch.Words[index], AvatarSeed: seed, NameSelectedAt: now, NameChangeAvailableAt: anonymousnames.Anniversary(now)}
			if err := tx.Create(&p).Error; err != nil {
				return err
			}
			if err := tx.Create(&identity.Binding{OwnerID: owner, PersonaUID: uid, SelectionBatch: batchID, SelectionIndex: index}).Error; err != nil {
				return err
			}
		}
		result = Public(p)
		return nil
	})
	return result, err
}
func (s Service) SetDisabled(owner uint64, disabled bool) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := users.LockAnonymousOwnerTx(tx, owner); err != nil {
			return err
		}
		p, err := personaForOwner(tx, owner)
		if err != nil {
			return err
		}
		if p.GovernanceDisabled && !disabled {
			return ErrUnavailable
		}
		return tx.Model(&p).Update("disabled", disabled).Error
	})
}

// Profile privacy remains manageable during a publishing restriction. Only the
// current live human owner is accepted, and no caller-supplied persona UID is used.
func (s Service) SetShowContent(owner uint64, show bool) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := users.LockAnonymousOwnerForReadTx(tx, owner); err != nil {
			return err
		}
		p, err := personaForOwner(tx, owner)
		if err != nil {
			return err
		}
		return tx.Model(&p).Update("show_content", show).Error
	})
}

// Resolve validates an explicit write identity. An unavailable persona never falls back to member.
func (s Service) Resolve(owner uint64, choice string) (string, error) {
	if choice == "" || choice == "member" {
		return "", nil
	}
	if choice != "persona" {
		return "", ErrUnavailable
	}
	var uid string
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := users.LockAnonymousOwnerTx(tx, owner); err != nil {
			return err
		}
		p, err := personaForOwner(tx, owner)
		if err != nil {
			return ErrUnavailable
		}
		if p.Disabled || p.GovernanceDisabled {
			return ErrUnavailable
		}
		uid = p.UID
		return nil
	})
	return uid, err
}

// Cleanup never touches retained personas, bindings, audits or the active-day quota.
func (s Service) Cleanup() error {
	cutoff := s.Now().Add(-7 * 24 * time.Hour)
	if err := s.DB.Where("expires_at < ?", cutoff).Delete(&identity.Batch{}).Error; err != nil {
		return err
	}
	day, _ := anonymousnames.Day(cutoff)
	return s.DB.Where("day < ?", day).Delete(&identity.Quota{}).Error
}
func ErrorCode(err error) string {
	if errors.Is(err, ErrLocked) || errors.Is(err, ErrQuota) || errors.Is(err, ErrCandidate) || errors.Is(err, ErrUnavailable) || errors.Is(err, ErrInvalidParams) {
		return err.Error()
	}
	return "operation.failed"
}
func ValidateReason(reason string) bool {
	return len([]rune(strings.TrimSpace(reason))) >= 1 && len([]rune(reason)) <= 512
}

func ValidatePublicUID(uid string) bool {
	if len(uid) != 32 {
		return false
	}
	for _, char := range uid {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
