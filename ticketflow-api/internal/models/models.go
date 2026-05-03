package models

import (
	"time"

	"github.com/google/uuid"
)

// Визначення користувацьких типів для перелічуваних значень (Enums)
type Role string
type TicketPriority string
type TicketStatus string

// Константи для ролей користувачів
const (
	RoleClient   Role = "client"
	RoleOperator Role = "operator"
	RoleEngineer Role = "engineer"
	RoleAdmin    Role = "admin"
)

// Константи для пріоритетів заявок
const (
	PriorityHigh   TicketPriority = "High"
	PriorityMedium TicketPriority = "Medium"
	PriorityLow    TicketPriority = "Low"
)

// Константи для статусів заявок
const (
	StatusNew        TicketStatus = "New"
	StatusInProgress TicketStatus = "In Progress"
	StatusResolved   TicketStatus = "Resolved"
	StatusClosed     TicketStatus = "Closed"
	StatusReopened   TicketStatus = "Reopened"
)

// Organization представляє клієнтську компанію для multi-tenant архітектури
type Organization struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string    `gorm:"type:varchar(255);unique;not null"`
	CreatedAt time.Time
	// Зв'язки
	Users   []User   `gorm:"foreignKey:OrganizationID"`
	Tickets []Ticket `gorm:"foreignKey:OrganizationID"`
}

// User представляє обліковий запис у системі
type User struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OrganizationID uuid.UUID `gorm:"type:uuid"`
	Email          string    `gorm:"type:varchar(255);unique;not null"`
	PasswordHash   string    `gorm:"type:varchar(255);not null"`
	Role           Role      `gorm:"type:user_role;not null"`
	FirstName      string    `gorm:"type:varchar(100);not null"`
	LastName       string    `gorm:"type:varchar(100);not null"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Ticket представляє заявку клієнта із ключовими полями для SLA
type Ticket struct {
	ID             uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index"`
	CreatorID      uuid.UUID      `gorm:"type:uuid;not null"`
	AssigneeID     *uuid.UUID     `gorm:"type:uuid"` // Вказівник, оскільки може бути NULL
	Title          string         `gorm:"type:varchar(255);not null"`
	Description    string         `gorm:"type:text;not null"`
	Status         TicketStatus   `gorm:"type:ticket_status;default:'New';index"`
	Priority       TicketPriority `gorm:"type:ticket_priority;default:'Medium'"`
	CreatedAt      time.Time      `gorm:"index"`
	ResolvedAt     *time.Time     // Важливе поле для розрахунку SLA
	UpdatedAt      time.Time

	// Навігаційні властивості GORM
	Creator     User            `gorm:"foreignKey:CreatorID"`
	Assignee    *User           `gorm:"foreignKey:AssigneeID"`
	Attachments []Attachment    `gorm:"foreignKey:TicketID"`
	Comments    []TicketComment `gorm:"foreignKey:TicketID"`
}

// Attachment описує файл, прикріплений до заявки
type Attachment struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TicketID    uuid.UUID `gorm:"type:uuid;not null"`
	UploaderID  uuid.UUID `gorm:"type:uuid;not null"`
	FileName    string    `gorm:"type:varchar(255);not null"`
	FilePath    string    `gorm:"type:varchar(500);not null"`
	FileSize    int       `gorm:"not null"` // Контролюється CHECK обмеженням у БД (макс 5 МБ)
	ContentType string    `gorm:"type:varchar(50);not null"`
	CreatedAt   time.Time
}

// TicketComment представляє коментар до заявки,
// що забезпечує комунікацію між учасниками процесу вирішення
type TicketComment struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TicketID  uuid.UUID `gorm:"type:uuid;not null;index"`
	AuthorID  uuid.UUID `gorm:"type:uuid;not null"`
	Message   string    `gorm:"type:text;not null"`
	CreatedAt time.Time

	// Навігаційні властивості GORM
	Ticket Ticket `gorm:"foreignKey:TicketID"`
	Author User   `gorm:"foreignKey:AuthorID"`
}
