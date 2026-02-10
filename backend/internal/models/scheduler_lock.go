package models

import "time"

// SchedulerLock represents a distributed lock for scheduled tasks
type SchedulerLock struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	LockName  string    `gorm:"uniqueIndex:idx_lock_name_key;size:100;not null" json:"lock_name"`
	LockKey   string    `gorm:"uniqueIndex:idx_lock_name_key;size:100;not null" json:"lock_key"`
	LockedBy  string    `gorm:"size:100" json:"locked_by"`
	LockedAt  time.Time `json:"locked_at"`
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
}

func (SchedulerLock) TableName() string { return "scheduler_locks" }
