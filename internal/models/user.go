package models

import "time"

type User struct {
	ID          string     `json:"id"          db:"id"`
	Email       string     `json:"email"       db:"email"`
	Username    string     `json:"username"    db:"username"`
	Password    string     `json:"-"           db:"password"`
	Age         int        `json:"age"         db:"age"`
	Description string     `json:"description" db:"description"`
	CreatedAt   time.Time  `json:"created_at"  db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"  db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"  db:"deleted_at"`
}
