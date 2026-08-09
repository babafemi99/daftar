package model

import "time"

const RefreshSessionCollection = "refresh_sessions"

type RefreshSession struct {
	ID           string     `bson:"_id"`
	UserID       string     `bson:"userId"`
	TokenHash    string     `bson:"tokenHash"`
	FamilyID     string     `bson:"familyId"`
	CreatedAt    time.Time  `bson:"createdAt"`
	ExpiresAt    time.Time  `bson:"expiresAt"`
	RevokedAt    *time.Time `bson:"revokedAt,omitempty"`
	ReplacedByID *string    `bson:"replacedById,omitempty"`
}
