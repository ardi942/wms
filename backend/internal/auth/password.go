package auth

import "wms/backend/internal/user"

func HashPassword(plain string) (string, error) { return user.HashPassword(plain) }

func VerifyPassword(hash, plain string) bool { return user.VerifyPassword(hash, plain) }
