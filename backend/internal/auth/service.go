package auth

import (
	"context"
	"strconv"
	"time"

	sharedauth "github.com/panadolextra91/myiu-lite/backend/internal/shared/auth"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo *Repository
	cfg  config.Config
}

func NewService(repo *Repository, cfg config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

func (s *Service) Login(ctx context.Context, username, password string) (db.User, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return db.User{}, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return db.User{}, ErrInvalidCredentials
	}

	return user, nil
}

func (s *Service) GetUser(ctx context.Context, userID int64) (db.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

func (s *Service) ChangePassword(ctx context.Context, userID int64, current, newPass, confirm string) error {
	if newPass != confirm {
		return ErrConfirmMismatch
	}

	if len(newPass) < 6 {
		return ErrTooShort
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(current))
	if err != nil {
		return ErrCurrentPasswordWrong
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPass))
	if err == nil {
		return ErrSameAsCurrent
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), 12)
	if err != nil {
		return err
	}

	return s.repo.UpdatePasswordAndStamp(ctx, userID, string(hash))
}

func (s *Service) Refresh(ctx context.Context, refreshToken string, secret string) (string, string, int64, error) {
	claims, err := sharedauth.Parse([]byte(secret), refreshToken)
	if err != nil {
		return "", "", 0, ErrInvalidCredentials
	}

	if claims.Type != "refresh" {
		return "", "", 0, ErrInvalidCredentials
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return "", "", 0, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return "", "", 0, ErrInvalidCredentials
	}

	if claims.IssuedAt != nil && claims.IssuedAt.Time.Before(user.PasswordChangedAt.Time) {
		return "", "", 0, ErrInvalidCredentials
	}

	newAccess, err := sharedauth.Mint([]byte(secret), user.ID, string(user.Role), "access", 15*time.Minute)
	if err != nil {
		return "", "", 0, err
	}

	return newAccess, string(user.Role), user.ID, nil
}
