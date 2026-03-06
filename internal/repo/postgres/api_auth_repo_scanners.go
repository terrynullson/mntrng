package postgres

import "github.com/terrynullson/hls_mntrng/internal/domain"

func scanUserRecord(scanner interface {
	Scan(dest ...interface{}) error
}) (domain.UserRecord, error) {
	var item domain.UserRecord
	err := scanner.Scan(
		&item.ID,
		&item.CompanyID,
		&item.Email,
		&item.Login,
		&item.PasswordHash,
		&item.Role,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return domain.UserRecord{}, err
	}
	return item, nil
}

func scanAuthSessionUser(scanner interface {
	Scan(dest ...interface{}) error
}) (domain.AuthSessionUser, error) {
	var item domain.AuthSessionUser
	err := scanner.Scan(
		&item.Session.ID,
		&item.Session.UserID,
		&item.Session.CompanyID,
		&item.Session.AccessTokenHash,
		&item.Session.RefreshTokenHash,
		&item.Session.AccessExpiresAt,
		&item.Session.RefreshExpiresAt,
		&item.Session.RevokedAt,
		&item.Session.CreatedAt,
		&item.Session.UpdatedAt,
		&item.User.ID,
		&item.User.CompanyID,
		&item.User.Email,
		&item.User.Login,
		&item.User.Role,
		&item.User.Status,
		&item.User.CreatedAt,
		&item.User.UpdatedAt,
	)
	if err != nil {
		return domain.AuthSessionUser{}, err
	}
	return item, nil
}
